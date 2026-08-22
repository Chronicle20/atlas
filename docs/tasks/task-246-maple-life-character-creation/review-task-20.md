# Review: Task 20 — seed the `mapleLife` class table

**Commit reviewed:** `a7d6d0b80` (`feat(configurations): seed the Maple Life class table`), single commit.
**Brief:** `.superpowers/sdd/plan/task-20-brief.md`
**Report:** `.superpowers/sdd/plan/task-20-report.md`
**Source of truth:** `docs/tasks/task-246-maple-life-character-creation/maple-life-content.md`
**Struct reviewed against:** `services/atlas-configurations/atlas.com/configurations/templates/maplelife/rest.go` (Task 19, `f43e442`)

## Scope confirmed

`git show --stat a7d6d0b80`:

```
services/atlas-configurations/seed-data/templates/template_gms_83_1.json | 568 ++++++++
services/atlas-configurations/seed-data/templates/template_gms_87_1.json | 568 ++++++++
services/atlas-configurations/seed-data/templates/template_gms_92_1.json | 568 ++++++++
services/atlas-configurations/seed-data/templates/template_gms_95_1.json | 568 ++++++++
4 files changed, 2272 insertions(+)
```

Matches the brief's file list exactly — four in-scope templates, insertions only, no deletions. Confirmed by direct diff, not by trusting the report.

## 1. Every value against `maple-life-content.md`

Extracted the `mapleLife` block from all four templates and diffed them byte-for-byte (JSON re-serialized, sorted keys): **`template_gms_83_1.json` == `template_gms_87_1.json` == `template_gms_92_1.json` == `template_gms_95_1.json`**. The content doc records no per-version divergence for anything in §1/§3/§5, so identical blocks across all four is correct, not an oversight.

### `looks` (§1) — PASS

Both gender rows (`gender: 0` male, `gender: 1` female) match §1's table exactly:
- male: `faces=[20000,20001,20002]`, `hairs=[30030,30020,30000]`, `hairColors=[0,7,3,2]`, `skinColors=[0,1,2,3]`
- female: `faces=[21000,21001,21002]`, `hairs=[31000,31040,31050]`, `hairColors=[0,7,3,2]`, `skinColors=[0,1,2,3]`

Slots 4-7 (tops/bottoms/shoes/weapons, "not on the wire") correctly omitted from `looks` — the schema doesn't carry them there anyway.

### `classes` (§3/§5) — 10 rows checked field-by-field, PASS

| ordinal | jobId | level | mapId | str/dex/int/luk | hp/mp | ap | sp | spSkillId | meso |
|---|---|---|---|---|---|---|---|---|---|
| 0 Warrior | 100 ✓ | 30 ✓ | 102000000 ✓ | 35/4/4/4 ✓ | 804/150 ✓ | 114 ✓ | `87,0×9` ✓ | 1000001 ✓ | 100000 ✓ |
| 1 Magician | 200 ✓ | 30 ✓ | 101000000 ✓ | 4/4/20/4 ✓ | 398/672 ✓ | 129 ✓ | `87,0×9` ✓ | 2000001 ✓ | 100000 ✓ |
| 2 Bowman | 300 ✓ | 30 ✓ | 100000000 ✓ | 4/25/4/4 ✓ | 688/440 ✓ | 124 ✓ | `87,0×9` ✓ | absent ✓ | 100000 ✓ |
| 3 Thief | 400 ✓ | 30 ✓ | 103000000 ✓ | 4/25/4/4 ✓ | 688/440 ✓ | 124 ✓ | `87,0×9` ✓ | absent ✓ | 100000 ✓ |
| 4 Pirate | 500 ✓ | 30 ✓ | 120000000 ✓ | 4/20/4/4 ✓ | 775/**599** ✓ | 129 ✓ | `87,0×9` ✓ | absent ✓ | 100000 ✓ |

Each row present for both genders (10 total). All values traced against §3's job-id table, §5.1's town-map table, §3's stat-floor table, §5.3(a)'s midpoint table, and §3's AP-remainder table. No transcription drift found.

**Equipment**, checked per class:
- top/bottom/shoes: male `1040002`/`1060002`/`1072001`, female `1041002`/`1061002`/`1072001` on every row — matches §3's generic-equipment table.
- weapon: Warrior `1302077`, Magician `1372043`, Bowman `1452051`, Thief `1332063` — each a single entry, matches §3's weapon table.
- Pirate: **both** `1482000` and `1492000` present as two separate `EquipmentEntry` list items — matches §5.2's ruling exactly.

**Inventory**, identical across all 10 rows: `2000002`×100, `2000006`×100, `3010000`×1 — matches §3's fixed package.

## 2. The three user rulings

1. **HP/MP midpoint, skill-excluded** — every row's `hp`/`mp` matches §5.3(a)'s midpoint table verbatim (804/150, 398/672, 688/440, 688/440, 775/599). No Warrior/Magician SP-skill contribution baked in — correctly deferred to Task 22 per §5.3(b). PASS.
2. **Starting maps = town maps** — 102000000 (Perion), 101000000 (Ellinia), 100000000 (Henesys), 103000000 (Kerning City), 120000000 (Nautilus Harbor) — matches §5.1's table, not the earlier job-advancement-NPC candidate ids. PASS.
3. **Pirate seeds both weapons; 599.5 rounds down to 599** — confirmed above; `mp: 599` is present for Pirate in both gender rows (verified via direct Python read of the committed JSON, not the report's claim). PASS.

## 3. Decode-correctness against `rest.go` (Task 19)

Checked every JSON key against `RestModel`/`LookOptions`/`ClassEntry`/`StatBlock`/`EquipmentEntry`/`InventoryEntry`'s tags field by field:

- `looks[].gender/faces/hairs/hairColors/skinColors` — all present, no extra/missing keys.
- `classes[].ordinal/gender/jobId/level/mapId/stats/ap/sp/spSkillId/meso/equipment/inventory` — all present with correct types (`sp` seeded as the string `"87,0,0,0,0,0,0,0,0,0"`, matching the `SP string` tag; `spSkillId` omitted, not zeroed, for ordinals 2-4, matching the `omitempty` tag).
- `stats.{str,dex,int,luk,hp,mp}` — all six keys present on every row.
- `equipment[].{templateId,useAverageStats}` and `inventory[].{templateId,quantity}` — correct shape.

Ran `go build ./... && go test -count=1 ./...` in `services/atlas-configurations/atlas.com/configurations` fresh (not relying on the report's quoted run): full build and every package passes, including `atlas-configurations/socket`. `TestValidate_AcceptsEverySeedTemplate` passes with `total == 3329` in this worktree state — the corpus-size drift the report flagged as a known pre-existing failure is not present at HEAD; verified this task's own diff adds no socket-handler/writer bindings (`mapleLife` seed data has none), so this task did not touch that count either way. Consistent with the controller's note — not re-raised as a finding.

Top-level `RestModel` key match: `templates/rest.go:23` and `tenants/rest.go:23` both declare `MapleLife maplelife.RestModel \`json:"mapleLife"\``, matching the seeded top-level key exactly.

## 4. Blast radius

- `template_gms_84_1.json`: confirmed via direct JSON parse (`"mapleLife" in d` → `False`) — no block, VERSION-ABSENT as required.
- The five out-of-scope templates: not present in `git show --stat a7d6d0b80` at all — confirmed from the commit diff itself, not the report's assertion.

## 5. `EquipmentEntry.UseAverageStats` ruling

The implementer's flagged concern: the content doc supplies no value for this schema flag; they set it `true` everywhere, citing `characters.presets[].attributes.equipment` as precedent.

Verified the precedent directly: swept every `characters.presets[].attributes.equipment[].useAverageStats` value across all four in-scope templates (141-151 equipment entries per file). **Every single value across all four files is `true`; zero occurrences of `false`.** The precedent claim is accurate, not cherry-picked.

**Ruling: `true` is correct.** It is the uniform existing convention for every seeded equipment entry in these files (both admin-preset and, now, Maple Life starter gear), it does not affect item id/stat/quantity content (only whether the item rolls average vs. random stats on grant), and there is no competing convention anywhere in the touched files to contradict it. Not a blocking concern.

## Findings

No blocking findings. No non-blocking findings beyond what the implementer already self-flagged and disposed of correctly (§2's UNCONFIRMED ordinal order — correctly carried forward as shipped, not re-litigated; §4 item 2's top/bottom/shoes synthesis — correctly not re-derived, out of this task's scope).

## Not evaluable

- None. The full review surface (four seeded files, the struct they decode into, the commit diff, and the content doc) was directly inspectable.

## Verdict rationale

Every one of the ten class rows across the four in-scope templates matches `maple-life-content.md` exactly, field by field. All three user rulings (HP/MP midpoint, town maps, Pirate dual-weapon + MP round-down) were followed precisely. The block decodes cleanly against Task 19's struct (key-by-key check, plus a fresh, non-cached `go test` run confirming it). `template_gms_84_1.json` and the five out-of-scope templates are untouched, confirmed from the diff. The one flagged data-shape decision (`useAverageStats: true`) is backed by a verified, uniform, unbroken precedent and is the right call for starter gear.
