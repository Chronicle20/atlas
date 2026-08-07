# Cygnus Knights original GMS release anchor (OQ-3)

**Pinned: GMS v0.73 (Jul 29, 2009).**

Source: meymink patch log
(`https://raw.githubusercontent.com/meymink/Maplestory-Patch-Logs/master/README.md`,
fetched 2026-07-30, HTTP 200), entry at line ~3943-3956:

```
0.73 (Jul 29, 2009)

- Cygnus Knights Class
- Maker Skill
- Blessing of the Fairy
- Vicious' Hammer Item
- New Skin Colors (Green and Pink)
- Elemental Wand Changes
- Many New Quests
- Maple Trading System (MTS) Tax change
- Bug Fix
```

This is not a provisioned column (provisioned columns are gms
{12,48,61,72,79,83,84,87,92,95} + jms 185; there is no gms 73). It anchors
the release/unreleased boundary that the *provisioned* neighbors straddle:

- **gms 61** (v0.61, Oct 15, 2008): no Cygnus job WZ data at all. The full
  job list for this tenant totals 44 jobs; the highest job id present is
  `910`. Nothing in the `1000`+ range exists.
- **gms 72** (v0.72, Jun 24, 2009 — before the v0.73 Cygnus release):
  Cygnus job WZ data **is present** as an unreleased stub. Job `1000`
  (Cygnus Noblesse) exists with a full skill roster, e.g. skill `10001000`
  resolves to the untranslated Korean name "달팽이 세마리" ("Three
  Snails"). Jobs `1100/1110/1111/1112` (Dawn Warrior tree), `1200/1210/1211/1212`
  (Blaze Wizard tree), `1300/1310/1311/1312` (Wind Archer tree),
  `1400/1410/1411/1412` (Night Walker tree), `1500/1510/1511/1512`
  (Thunder Breaker tree) are likewise present in the v72 tenant's job list
  (job list total: 66, up from 44 at v61).
- **gms 83** (v0.83, Feb 22, 2010 — after the v0.73 Cygnus release):
  the same skill `10001000` now resolves to the English name "Three
  Snails" — released.

So within the provisioned set, the Cygnus release boundary sits between
**gms 72 (WZ stub, unreleased)** and **gms 79/83/84/87/92/95 (released)** —
gms 79 was not individually re-checked for job 1000's skill names in this
audit (job `1000`'s presence in gms 79's job list was not confirmed;
`availability.csv` marks gms 79 released=true on the v0.73 anchor date
alone, `72 < 73 < 79`, not on an independent gms_79 WZ check — flagged here
for transparency, not asserted as independently WZ-verified at gms_79
specifically).

This value is fully confirmed from the meymink log; nothing here is marked
`UNVERIFIED`.
