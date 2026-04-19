const JOB_BITS: Array<{ bit: number; name: string }> = [
  { bit: 1, name: "Warrior" },
  { bit: 2, name: "Magician" },
  { bit: 4, name: "Bowman" },
  { bit: 8, name: "Thief" },
  { bit: 16, name: "Pirate" },
];

export function formatReqJob(reqJob: number): string[] {
  if (!reqJob) return [];
  return JOB_BITS.filter(({ bit }) => (reqJob & bit) !== 0).map(({ name }) => name);
}
