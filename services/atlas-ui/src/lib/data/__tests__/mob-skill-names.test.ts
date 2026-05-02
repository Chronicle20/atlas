import { getMobSkillCanonicalName } from '../mob-skill-names';

// Pin a representative subset of the mob-skill name mapping. The table mirrors
// `libs/atlas-constants/monster/skill.go` line-for-line — these assertions
// guard against the previous bug where ids 110-129 were shifted (122 displayed
// as "Banish" when it is actually Weakness; real Banish at 129 was unmapped).
describe('getMobSkillCanonicalName', () => {
  it('returns the canonical name for player-targeting debuffs', () => {
    expect(getMobSkillCanonicalName(120)).toBe('Seal');
    expect(getMobSkillCanonicalName(121)).toBe('Darkness');
    expect(getMobSkillCanonicalName(122)).toBe('Weakness');
    expect(getMobSkillCanonicalName(123)).toBe('Stun');
    expect(getMobSkillCanonicalName(124)).toBe('Curse');
    expect(getMobSkillCanonicalName(125)).toBe('Poison');
    expect(getMobSkillCanonicalName(126)).toBe('Slow');
    expect(getMobSkillCanonicalName(127)).toBe('Dispel');
    expect(getMobSkillCanonicalName(128)).toBe('Seduce');
    expect(getMobSkillCanonicalName(129)).toBe('Banish');
  });

  it('distinguishes single-target vs AoE buff variants', () => {
    expect(getMobSkillCanonicalName(100)).toBe('Weapon Attack Up');
    expect(getMobSkillCanonicalName(110)).toBe('Weapon Attack Up (AoE)');
    expect(getMobSkillCanonicalName(101)).toBe('Magic Attack Up');
    expect(getMobSkillCanonicalName(111)).toBe('Magic Attack Up (AoE)');
  });

  it('returns the canonical name for immunities and counters', () => {
    expect(getMobSkillCanonicalName(140)).toBe('Physical Immune');
    expect(getMobSkillCanonicalName(141)).toBe('Magic Immune');
    expect(getMobSkillCanonicalName(143)).toBe('Physical Counter');
    expect(getMobSkillCanonicalName(144)).toBe('Magic Counter');
  });

  it('returns the canonical name for Heal and Summon', () => {
    expect(getMobSkillCanonicalName(114)).toBe('Heal');
    expect(getMobSkillCanonicalName(200)).toBe('Summon');
  });

  it('returns undefined for unknown ids so the chip falls back to the raw id', () => {
    expect(getMobSkillCanonicalName(0)).toBeUndefined();
    expect(getMobSkillCanonicalName(99)).toBeUndefined();
    expect(getMobSkillCanonicalName(999)).toBeUndefined();
  });
});
