import { getJobNameById } from "@/lib/jobs";

export type Archetype = "Adventurer" | "Cygnus" | "Legend" | "Admin";

export interface JobNode {
  jobId: number; // key for /api/data/jobs/{id}/skills
  minMajorVersion: number; // FR-2.2a version gate
}
export interface ClassNode {
  name: string;
  jobs: JobNode[];
}
export interface ArchetypeNode {
  name: Archetype;
  classes: ClassNode[];
}

/** Display name for a leaf — reuses jobNameMap, never duplicates it. */
export function jobNodeName(node: JobNode): string {
  return getJobNameById(node.jobId) ?? `Job ${node.jobId}`;
}

// minMajorVersion basis:
//  - 83: v83 baseline — all Adventurer jobs, GM/Super GM, and Maple Leaf
//        Brigadier exist in v83 data (confirmed by Task 7 probe).
//  - Evan (2001/22xx) = 84  — reference_maplestory_version_timeline (Evan ≈ v84).
//  - Aran (2000/21xx) = 88  — timeline floor (Aran predates Dual Blade ≈ v88);
//        confirm/adjust via Task 7 probe.
//  - Cygnus (1000–1512) = 92 — Knights of Cygnus; floor only needs to exceed 83
//        so it hides on the v83 baseline. Confirm via Task 7 probe.
const ADV = 83;
const ADMIN = 83;
const CYGNUS = 92;
const ARAN = 88;
const EVAN = 84;

export const JOB_HIERARCHY: ArchetypeNode[] = [
  {
    name: "Adventurer",
    classes: [
      { name: "Beginner", jobs: [{ jobId: 0, minMajorVersion: ADV }] },
      {
        name: "Warrior",
        jobs: [100, 110, 111, 112, 120, 121, 122, 130, 131, 132].map((jobId) => ({ jobId, minMajorVersion: ADV })),
      },
      {
        name: "Magician",
        jobs: [200, 210, 211, 212, 220, 221, 222, 230, 231, 232].map((jobId) => ({ jobId, minMajorVersion: ADV })),
      },
      {
        name: "Bowman",
        jobs: [300, 310, 311, 312, 320, 321, 322].map((jobId) => ({ jobId, minMajorVersion: ADV })),
      },
      {
        name: "Thief",
        jobs: [400, 410, 411, 412, 420, 421, 422].map((jobId) => ({ jobId, minMajorVersion: ADV })),
      },
      {
        name: "Pirate",
        jobs: [500, 510, 511, 512, 520, 521, 522].map((jobId) => ({ jobId, minMajorVersion: ADV })),
      },
      { name: "Special", jobs: [{ jobId: 800, minMajorVersion: ADV }] },
    ],
  },
  {
    name: "Cygnus",
    classes: [
      { name: "Noblesse", jobs: [{ jobId: 1000, minMajorVersion: CYGNUS }] },
      { name: "Dawn Warrior", jobs: [1100, 1110, 1111, 1112].map((jobId) => ({ jobId, minMajorVersion: CYGNUS })) },
      { name: "Blaze Wizard", jobs: [1200, 1210, 1211, 1212].map((jobId) => ({ jobId, minMajorVersion: CYGNUS })) },
      { name: "Wind Archer", jobs: [1300, 1310, 1311, 1312].map((jobId) => ({ jobId, minMajorVersion: CYGNUS })) },
      { name: "Night Walker", jobs: [1400, 1410, 1411, 1412].map((jobId) => ({ jobId, minMajorVersion: CYGNUS })) },
      { name: "Thunder Breaker", jobs: [1500, 1510, 1511, 1512].map((jobId) => ({ jobId, minMajorVersion: CYGNUS })) },
    ],
  },
  {
    name: "Legend",
    classes: [
      { name: "Legend", jobs: [{ jobId: 2000, minMajorVersion: ARAN }] },
      { name: "Aran", jobs: [2100, 2110, 2111, 2112].map((jobId) => ({ jobId, minMajorVersion: ARAN })) },
      {
        name: "Evan",
        jobs: [2001, 2200, 2210, 2211, 2212, 2213, 2214, 2215, 2216, 2217, 2218].map((jobId) => ({ jobId, minMajorVersion: EVAN })),
      },
    ],
  },
  {
    name: "Admin",
    classes: [{ name: "GM", jobs: [900, 910].map((jobId) => ({ jobId, minMajorVersion: ADMIN })) }],
  },
];

/**
 * Prune job-tiers above the tenant's major version, then drop classes with no
 * surviving jobs and archetypes with no surviving classes (FR-2.3). Pure — does
 * not mutate the source tree.
 */
export function filterHierarchy(tree: ArchetypeNode[], major: number): ArchetypeNode[] {
  return tree
    .map((arch) => ({
      name: arch.name,
      classes: arch.classes
        .map((cls) => ({
          name: cls.name,
          jobs: cls.jobs.filter((j) => j.minMajorVersion <= major),
        }))
        .filter((cls) => cls.jobs.length > 0),
    }))
    .filter((arch) => arch.classes.length > 0);
}
