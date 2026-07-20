// Curated jobId → display name. atlas-data exposes no job-name endpoint
// (jobs.service only serves /jobs/{id}/skills), so names are a client map.
// Covers seed-data ids + common advances; the backend is the validator of
// record, so any unmapped id renders as `Job <id>` and remains selectable.
export const PRESET_JOBS: { id: number; name: string }[] = [
  { id: 0, name: "Beginner" },
  { id: 100, name: "Warrior" },
  { id: 110, name: "Fighter" },
  { id: 111, name: "Crusader" },
  { id: 112, name: "Hero" },
  { id: 120, name: "Page" },
  { id: 121, name: "White Knight" },
  { id: 122, name: "Paladin" },
  { id: 130, name: "Spearman" },
  { id: 131, name: "Dragon Knight" },
  { id: 132, name: "Dark Knight" },
  { id: 200, name: "Magician" },
  { id: 210, name: "Fire/Poison Wizard" },
  { id: 211, name: "Fire/Poison Mage" },
  { id: 212, name: "Fire/Poison Archmage" },
  { id: 220, name: "Ice/Lightning Wizard" },
  { id: 221, name: "Ice/Lightning Mage" },
  { id: 222, name: "Ice/Lightning Archmage" },
  { id: 230, name: "Cleric" },
  { id: 231, name: "Priest" },
  { id: 232, name: "Bishop" },
  { id: 300, name: "Bowman" },
  { id: 310, name: "Hunter" },
  { id: 311, name: "Ranger" },
  { id: 312, name: "Bowmaster" },
  { id: 320, name: "Crossbowman" },
  { id: 321, name: "Sniper" },
  { id: 322, name: "Marksman" },
  { id: 400, name: "Thief" },
  { id: 410, name: "Assassin" },
  { id: 411, name: "Hermit" },
  { id: 412, name: "Night Lord" },
  { id: 420, name: "Bandit" },
  { id: 421, name: "Chief Bandit" },
  { id: 422, name: "Shadower" },
  { id: 500, name: "Pirate" },
  { id: 510, name: "Brawler" },
  { id: 511, name: "Marauder" },
  { id: 512, name: "Buccaneer" },
  { id: 520, name: "Gunslinger" },
  { id: 521, name: "Outlaw" },
  { id: 522, name: "Corsair" },
  { id: 900, name: "GM" },
  { id: 910, name: "SuperGM" },
];

const NAME_BY_ID = new Map(PRESET_JOBS.map((j) => [j.id, j.name]));

export function jobLabel(id: number): string {
  return NAME_BY_ID.get(id) ?? `Job ${id}`;
}
