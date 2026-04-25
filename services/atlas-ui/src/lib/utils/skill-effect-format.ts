export interface Statup {
  type: string;
  amount: number;
}

export function formatDurationMs(ms: number): string {
  if (!ms || ms <= 0) return "";
  const s = Math.round(ms / 1000);
  return ` for ${s}s`;
}

const LABELS: Record<string, string> = {
  WeaponAttack: "Weapon Attack",
  WeaponDefense: "Weapon Defense",
  MagicAttack: "Magic Attack",
  MagicDefense: "Magic Defense",
  Accuracy: "Accuracy",
  Avoidability: "Avoidability",
  Speed: "Speed",
  Jump: "Jump",
  Hp: "HP",
  Mp: "MP",
  MaxHp: "Max HP",
  MaxMp: "Max MP",
};

export function formatStatup(s: Statup, durationMs: number): string {
  const label = LABELS[s.type];
  if (label) return `+${s.amount} ${label}${formatDurationMs(durationMs)}`;
  return `${s.type}: +${s.amount}`;
}
