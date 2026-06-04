import { describe, expect, it } from 'vitest';
import fixture from './loadout-hashes.json';
import {
  canonicalLoadoutString,
  loadoutHash,
  generateCharacterUrl,
  filterEquipment,
  resolveGender,
  type Stance,
} from '../characterRender.service';

interface FixtureRow {
  tenant: string;
  region: string;
  majorVersion: number;
  minorVersion: number;
  skin: number;
  hair: number;
  face: number;
  stance: string;
  frame: number;
  resize: number;
  items: number[];
  gender: number;
  canonical: string;
  expectedHash: string;
}

describe('characterRender canonical+hash parity', () => {
  for (const row of (fixture as { rows: FixtureRow[] }).rows) {
    it(`row ${row.tenant} ${row.stance} matches canonical`, () => {
      const canonical = canonicalLoadoutString(
        row.tenant, row.region, row.majorVersion, row.minorVersion,
        row.skin, row.hair, row.face,
        row.stance as Stance, row.frame, row.resize, row.items, row.gender,
      );
      expect(canonical).toBe(row.canonical);
      expect(loadoutHash(canonical)).toBe(row.expectedHash);
    });
  }
});

describe('filterEquipment', () => {
  it('drops mount, pet, and cash slots', () => {
    const out = filterEquipment({
      '-1': 1002357,
      '-11': 1402024,
      '-14': 5000000,
      '-18': 1932000,
      '-19': 1932001,
      '-21': 1012000,
      '-101': 1002001,
      '-114': 1132001,
    });
    expect(out['-1']).toBe(1002357);
    expect(out['-11']).toBe(1402024);
    for (const slot of ['-14', '-18', '-19', '-21', '-101', '-114']) {
      expect(out[slot]).toBeUndefined();
    }
  });
});

describe('generateCharacterUrl', () => {
  it('builds the documented path/query shape', () => {
    const url = generateCharacterUrl(
      'tenant-a', 'GMS', 83, 1,
      { skin: 0, hair: 30030, face: 20000, equipment: { '-1': 1002357 } },
      { stance: 'stand1', frame: 0, resize: 2 },
    );
    expect(url.startsWith('/api/assets/tenant-a/GMS/83.1/character/')).toBe(true);
    expect(url).toMatch(/\/[a-f0-9]{16}\.png\?/);
    expect(url).toContain('skin=0');
    expect(url).toContain('hair=30030');
    expect(url).toContain('face=20000');
    expect(url).toContain('stance=stand1');
    expect(url).toContain('items=1002357');
  });

  it('sorts items so order does not change the URL', () => {
    const a = generateCharacterUrl('t', 'GMS', 83, 1,
      { skin: 0, hair: 30030, face: 20000, equipment: { '-1': 1442024, '-11': 1002357, '-5': 1402024 } },
      {});
    const b = generateCharacterUrl('t', 'GMS', 83, 1,
      { skin: 0, hair: 30030, face: 20000, equipment: { '-11': 1002357, '-5': 1402024, '-1': 1442024 } },
      {});
    expect(a).toBe(b);
  });
});

describe('resolveGender', () => {
  it('explicit 0/1 wins over face inference', () => {
    expect(resolveGender(0, 21000)).toBe(0);
    expect(resolveGender(1, 20000)).toBe(1);
  });
  it('infers female from 21xxx face', () => {
    expect(resolveGender(undefined, 21000)).toBe(1);
  });
  it('infers male from 20xxx face', () => {
    expect(resolveGender(undefined, 20000)).toBe(0);
  });
  it('non-positive / unknown face → male', () => {
    expect(resolveGender(undefined, 0)).toBe(0);
    expect(resolveGender(undefined, -5)).toBe(0);
    expect(resolveGender(undefined, 30030)).toBe(0);
  });
});

describe('canonicalLoadoutString gender', () => {
  it('appends resolved gender as the final field', () => {
    const c = canonicalLoadoutString('T', 'GMS', 83, 1, 0, 30030, 20000, 'stand1', 0, 2, [], 1);
    expect(c).toBe('T|GMS|83.1|0|30030|20000|stand1|0|2||1');
  });
});

describe('generateCharacterUrl gender', () => {
  it('emits a gender query param and a hash matching the canonical string', () => {
    const url = generateCharacterUrl(
      'tenant-a', 'GMS', 83, 1,
      { skin: 0, hair: 30030, face: 20000, equipment: {}, gender: 1 },
      { stance: 'stand1', frame: 0, resize: 2 },
    );
    expect(url).toContain('gender=1');
    const expected = loadoutHash(
      canonicalLoadoutString('tenant-a', 'GMS', 83, 1, 0, 30030, 20000, 'stand1', 0, 2, [], 1),
    );
    expect(url).toContain(`/${expected}.png?`);
  });

  it('male-face vs female-face empty loadout produce different hashes (no collision)', () => {
    const male = generateCharacterUrl('t', 'GMS', 83, 1,
      { skin: 0, hair: 30030, face: 20000, equipment: {} }, {});
    const female = generateCharacterUrl('t', 'GMS', 83, 1,
      { skin: 0, hair: 30030, face: 21000, equipment: {} }, {});
    const hashOf = (u: string) => u.match(/\/([a-f0-9]{16})\.png\?/)?.[1];
    expect(hashOf(male)).toBeDefined();
    expect(hashOf(female)).toBeDefined();
    expect(hashOf(male)).not.toBe(hashOf(female));
  });
});
