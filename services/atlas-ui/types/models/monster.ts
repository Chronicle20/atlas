export interface MonsterAttributes {
  name: string;
  hp: number;
  mp: number;
  experience: number;
  level: number;
  weapon_attack: number;
  weapon_defense: number;
  magic_attack: number;
  magic_defense: number;
  friendly: boolean;
  boss: boolean;
  undead: boolean;
  remove_after: number;
  explosive_reward: boolean;
  ffa_loot: boolean;
  buff_to_give: number;
  cp: number;
  remove_on_miss: boolean;
  changeable: boolean;
  first_attack: boolean;
  drop_period: number;
  tag_color: number;
  tag_background_color: number;
  fixed_stance: number;
  lose_items: MonsterLoseItem[];
  skills: MonsterSkill[];
  revives: number[];
}

export interface MonsterLoseItem {
  id: number;
  chance: number;
  x: number;
}

export interface MonsterSkill {
  id: number;
  level: number;
}

export interface MonsterData {
  id: string;
  type: string;
  attributes: MonsterAttributes;
}
