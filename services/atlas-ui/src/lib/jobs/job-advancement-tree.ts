export interface JobEntry {
  id: number;
  name: string;
  parent: number | null;
}

// Structural source of truth for the job advancement graph.
// Ported from the former lib/utils/job-tree.ts JOB_TREE, whose ids/names derive
// from libs/atlas-constants/job/constants.go::Jobs (v83 conventions). One
// intentional divergence: constants.go has GM (900) and Super GM (910) as
// roots, but in-game they present as an advancement line from Beginner, so
// this display graph parents 900 under 0 and 910 under 900 (task-182).
// Order per branch: branch leader -> 1st -> 2nd -> 3rd -> 4th.
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
  // Special / Admin. Maple Leaf Brigadier stays a standalone root. In-game, GM
  // and Super GM present as an advancement line from Beginner, so the DISPLAY
  // graph adopts Beginner > GM > Super GM — an intentional divergence from
  // libs/atlas-constants/job/constants.go, where 900/910 are roots (task-182).
  800: { id: 800, name: "Maple Leaf Brigadier", parent: null },
  900: { id: 900, name: "GM", parent: 0 },
  910: { id: 910, name: "Super GM", parent: 900 },
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

/** Every graph entry, ascending by id — the full selectable job set before tenant gating. */
export const JOB_LIST: JobEntry[] = Object.values(JOB_GRAPH).sort(
  (a, b) => a.id - b.id,
);

/**
 * Display name for a job id, sourced from the advancement graph so Aran, Evan
 * and Cygnus render by name everywhere (preset badges, class picker, rankings)
 * — not just the explorer classes the old curated list covered. Falls back to
 * `Job <id>` for an id the graph does not cover; the backend stays the
 * validator of record, so an unmapped id remains usable.
 */
export function jobName(id: number): string {
  return JOB_GRAPH[id]?.name ?? `Job ${id}`;
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
