import { describe, expect, it } from 'vitest';
import { generateQueryKey } from '../useCharacterImage';
import { generateCharacterUrl } from '@/services/api/characterRender.service';
import type { MapleStoryCharacterData } from '@/types/models/maplestory';

function makeCharacter(overrides: Partial<MapleStoryCharacterData> = {}): MapleStoryCharacterData {
  return {
    id: '1',
    name: 'Test',
    level: 1,
    jobId: 0,
    tenant: 'tenant-a',
    region: 'GMS',
    majorVersion: 83,
    minorVersion: 1,
    skinColor: 0,
    hair: 30030,
    face: 20000,
    gender: 0,
    equipment: {},
    ...overrides,
  } as MapleStoryCharacterData;
}

function urlHash(u: string): string | undefined {
  return u.match(/\/([a-f0-9]{16})\.png\?/)?.[1];
}

describe('useCharacterImage query-key vs URL hash parity', () => {
  for (const c of [makeCharacter(), makeCharacter({ face: 21000, gender: 1 })]) {
    it(`gender ${c.gender} face ${c.face} key hash equals URL hash`, () => {
      const keyHash = generateQueryKey(c)[1];
      const url = generateCharacterUrl(
        c.tenant, c.region, c.majorVersion, c.minorVersion,
        {
          skin: c.skinColor,
          hair: c.hair,
          face: c.face,
          equipment: Object.fromEntries(
            Object.entries(c.equipment).map(([k, v]) => [k, v as number]),
          ),
          gender: c.gender,
        },
        {},
      );
      expect(keyHash).toBe(urlHash(url));
    });
  }
});
