export interface JobEntry {
  id: number;
  name: string;
  parent: number | null;
}

// Structural source of truth for the job advancement graph.
// Ported verbatim from the former lib/utils/job-tree.ts JOB_TREE, whose ids/names
// derive from libs/atlas-constants/job/constants.go::Jobs (v83 conventions).
// Order per branch: branch leader (parent: null) -> 1st -> 2nd -> 3rd -> 4th.
export const JOB_GRAPH: Record<number, JobEntry> = {
  // Beginner branch
  0: { id: 0, name: "Beginner", parent: null },
  // Warrior
  100: { id: 100, name: "Warrior", parent: 0 },
  110: { id: 110, name: "Fighter", parent: 100 },
  111: { id: 111, name: "Crusader", parent: 110 },
  112: { id: 112, name: "Hero", parent: 111 },
  120: { id: 120, name: "Page", parent: 100 },
  121: { id: 121, name: "White Knight", parent: 120 },
  122: { id: 122, name: "Paladin", parent: 121 },
  130: { id: 130, name: "Spearman", parent: 100 },
  131: { id: 131, name: "Dragon Knight", parent: 130 },
  132: { id: 132, name: "Dark Knight", parent: 131 },
  // Magician
  200: { id: 200, name: "Magician", parent: 0 },
  210: { id: 210, name: "Wizard (F/P)", parent: 200 },
  211: { id: 211, name: "Mage (F/P)", parent: 210 },
  212: { id: 212, name: "Arch Mage (F/P)", parent: 211 },
  220: { id: 220, name: "Wizard (I/L)", parent: 200 },
  221: { id: 221, name: "Mage (I/L)", parent: 220 },
  222: { id: 222, name: "Arch Mage (I/L)", parent: 221 },
  230: { id: 230, name: "Cleric", parent: 200 },
  231: { id: 231, name: "Priest", parent: 230 },
  232: { id: 232, name: "Bishop", parent: 231 },
  // Bowman
  300: { id: 300, name: "Bowman", parent: 0 },
  310: { id: 310, name: "Hunter", parent: 300 },
  311: { id: 311, name: "Ranger", parent: 310 },
  312: { id: 312, name: "Bowmaster", parent: 311 },
  320: { id: 320, name: "Crossbowman", parent: 300 },
  321: { id: 321, name: "Sniper", parent: 320 },
  322: { id: 322, name: "Marksman", parent: 321 },
  // Thief
  400: { id: 400, name: "Rogue", parent: 0 },
  410: { id: 410, name: "Assassin", parent: 400 },
  411: { id: 411, name: "Hermit", parent: 410 },
  412: { id: 412, name: "Night Lord", parent: 411 },
  420: { id: 420, name: "Bandit", parent: 400 },
  421: { id: 421, name: "Chief Bandit", parent: 420 },
  422: { id: 422, name: "Shadower", parent: 421 },
  // Pirate
  500: { id: 500, name: "Pirate", parent: 0 },
  510: { id: 510, name: "Brawler", parent: 500 },
  511: { id: 511, name: "Marauder", parent: 510 },
  512: { id: 512, name: "Buccaneer", parent: 511 },
  520: { id: 520, name: "Gunslinger", parent: 500 },
  521: { id: 521, name: "Outlaw", parent: 520 },
  522: { id: 522, name: "Corsair", parent: 521 },
  // Special / Admin (standalone roots)
  800: { id: 800, name: "Maple Leaf Brigadier", parent: null },
  900: { id: 900, name: "GM", parent: null },
  910: { id: 910, name: "Super GM", parent: null },
  // Noblesse / Cygnus Knights
  1000: { id: 1000, name: "Noblesse", parent: null },
  1100: { id: 1100, name: "Dawn Warrior 1", parent: 1000 },
  1110: { id: 1110, name: "Dawn Warrior 2", parent: 1100 },
  1111: { id: 1111, name: "Dawn Warrior 3", parent: 1110 },
  1112: { id: 1112, name: "Dawn Warrior 4", parent: 1111 },
  1200: { id: 1200, name: "Blaze Wizard 1", parent: 1000 },
  1210: { id: 1210, name: "Blaze Wizard 2", parent: 1200 },
  1211: { id: 1211, name: "Blaze Wizard 3", parent: 1210 },
  1212: { id: 1212, name: "Blaze Wizard 4", parent: 1211 },
  1300: { id: 1300, name: "Wind Archer 1", parent: 1000 },
  1310: { id: 1310, name: "Wind Archer 2", parent: 1300 },
  1311: { id: 1311, name: "Wind Archer 3", parent: 1310 },
  1312: { id: 1312, name: "Wind Archer 4", parent: 1311 },
  1400: { id: 1400, name: "Night Walker 1", parent: 1000 },
  1410: { id: 1410, name: "Night Walker 2", parent: 1400 },
  1411: { id: 1411, name: "Night Walker 3", parent: 1410 },
  1412: { id: 1412, name: "Night Walker 4", parent: 1411 },
  1500: { id: 1500, name: "Thunder Breaker 1", parent: 1000 },
  1510: { id: 1510, name: "Thunder Breaker 2", parent: 1500 },
  1511: { id: 1511, name: "Thunder Breaker 3", parent: 1510 },
  1512: { id: 1512, name: "Thunder Breaker 4", parent: 1511 },
  // Legend / Aran
  2000: { id: 2000, name: "Legend", parent: null },
  2100: { id: 2100, name: "Aran 1", parent: 2000 },
  2110: { id: 2110, name: "Aran 2", parent: 2100 },
  2111: { id: 2111, name: "Aran 3", parent: 2110 },
  2112: { id: 2112, name: "Aran 4", parent: 2111 },
  // Evan (separate root per job/constants.go)
  2001: { id: 2001, name: "Evan", parent: null },
  2200: { id: 2200, name: "Evan 1", parent: 2001 },
  2210: { id: 2210, name: "Evan 2", parent: 2200 },
  2211: { id: 2211, name: "Evan 3", parent: 2210 },
  2212: { id: 2212, name: "Evan 4", parent: 2211 },
  2213: { id: 2213, name: "Evan 5", parent: 2212 },
  2214: { id: 2214, name: "Evan 6", parent: 2213 },
  2215: { id: 2215, name: "Evan 7", parent: 2214 },
  2216: { id: 2216, name: "Evan 8", parent: 2215 },
  2217: { id: 2217, name: "Evan 9", parent: 2216 },
  2218: { id: 2218, name: "Evan 10", parent: 2217 },
};

// Version-introduction floor per branch ROOT id. A node inherits its root's floor
// (gating is per-branch, never per-node). This is a DISPLAY-curation choice — the
// atlas-data /jobs/{id}/skills endpoint is NOT version-gated (live probe 2026-06-14)
// — so floors hide classes that did not exist in the live game at the tenant's
// version; they are not a data-availability gate.
//   0    Adventurers          — v83 baseline
//   800  Maple Leaf Brigadier  — v83 baseline (special)
//   900  GM                    — admin, always present
//   910  Super GM              — admin, always present
//   1000 Cygnus (Noblesse)     — reference_maplestory_version_timeline: KoC exist in v83
//   2000 Legend (Aran)         — product owner (PRD FR-8.1): Aran introduced v80
//   2001 Evan                  — reference_maplestory_version_timeline: Evan introduced v84
export const BRANCH_FLOORS: Record<number, number> = {
  0: 83, 800: 83, 900: 83, 910: 83, 1000: 83, 2000: 80, 2001: 84,
};

/** Branch root ids (parent === null), ascending. */
export const JOB_ROOTS: number[] = Object.values(JOB_GRAPH)
  .filter((e) => e.parent === null)
  .map((e) => e.id)
  .sort((a, b) => a - b);

/** Direct children of a node, ascending by id. */
export function childrenOf(id: number): number[] {
  return Object.values(JOB_GRAPH)
    .filter((e) => e.parent === id)
    .map((e) => e.id)
    .sort((a, b) => a - b);
}

/** Walk parent edges to the branch root. Returns the id itself if it is a root or unknown. */
export function rootOf(id: number): number {
  let cur = JOB_GRAPH[id];
  if (!cur) return id;
  while (cur.parent != null) {
    const next = JOB_GRAPH[cur.parent];
    if (!next) break;
    cur = next;
  }
  return cur.id;
}

/** Version floor for a node = its root's BRANCH_FLOORS entry (0 = fail-open if unfloored). */
export function floorOf(id: number): number {
  return BRANCH_FLOORS[rootOf(id)] ?? 0;
}

/** Root ids visible at the given tenant major version, ascending. */
export function visibleRoots(major: number): number[] {
  return JOB_ROOTS.filter((r) => floorOf(r) <= major);
}

/** Root -> node advancement path (inclusive), for breadcrumbs. */
export function jobTreePath(jobId: number): JobEntry[] {
  const path: JobEntry[] = [];
  let cur: JobEntry | undefined = JOB_GRAPH[jobId];
  while (cur) {
    path.unshift(cur);
    cur = cur.parent != null ? JOB_GRAPH[cur.parent] : undefined;
  }
  return path;
}
