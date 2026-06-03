import { sha256 } from 'js-sha256';
import type { Character } from '@/types/models/character';
import type { Asset } from '@/services/api/inventory.service';

export type Stance = 'stand1' | 'stand2' | 'walk1' | 'alert' | 'jump';

export interface CharacterLoadout {
  skin: number;
  hair: number;
  face: number;
  equipment: Record<string, number>;
  gender?: number;
}

export interface RenderOptions {
  stance?: Stance;
  frame?: number;
  resize?: number;
}

const CASH_SLOT_MIN = -114;
const CASH_SLOT_MAX = -101;
const FIXED_DROPPED_SLOTS = new Set([
  -14, // pet
  -18, -19, -20, // mount
  -21, -22, -23, -24, -25, -26, -27, -28, -29, -30, // pet rings
]);

export function filterEquipment(eq: Record<string, number>): Record<string, number> {
  const out: Record<string, number> = {};
  for (const [slot, id] of Object.entries(eq)) {
    const n = parseInt(slot, 10);
    if (FIXED_DROPPED_SLOTS.has(n)) continue;
    if (n >= CASH_SLOT_MIN && n <= CASH_SLOT_MAX) continue;
    out[slot] = id;
  }
  return out;
}

/**
 * Mirror of the Go service's ResolveGender. An explicit 0/1 wins; otherwise
 * infer from the face id via the v83 convention (face/1000)%10 === 1 ⇒ female.
 * A non-positive / unknown face resolves to male (0).
 */
export function resolveGender(gender: number | undefined, face: number): 0 | 1 {
  if (gender === 0 || gender === 1) return gender;
  if (face > 0 && Math.floor(face / 1000) % 10 === 1) return 1;
  return 0;
}

export function canonicalLoadoutString(
  tenant: string,
  region: string,
  major: number,
  minor: number,
  skin: number,
  hair: number,
  face: number,
  stance: Stance,
  frame: number,
  resize: number,
  items: number[],
  gender: number,
): string {
  const sorted = [...items].sort((a, b) => a - b);
  return [
    tenant, region, `${major}.${minor}`,
    skin, hair, face,
    stance, frame, resize,
    sorted.join(','),
    gender,
  ].join('|');
}

export function loadoutHash(canonical: string): string {
  return sha256(canonical).slice(0, 16);
}

export function generateCharacterUrl(
  tenant: string,
  region: string,
  major: number,
  minor: number,
  loadout: CharacterLoadout,
  options: RenderOptions = {},
): string {
  const opts: Required<RenderOptions> = {
    stance: options.stance ?? 'stand1',
    frame: options.frame ?? 0,
    resize: options.resize ?? 2,
  };
  const filtered = filterEquipment(loadout.equipment);
  const items = Object.values(filtered).sort((a, b) => a - b);
  const gender = resolveGender(loadout.gender, loadout.face);
  const canonical = canonicalLoadoutString(
    tenant, region, major, minor,
    loadout.skin, loadout.hair, loadout.face,
    opts.stance, opts.frame, opts.resize, items, gender,
  );
  const hash = loadoutHash(canonical);
  const params = new URLSearchParams({
    skin: String(loadout.skin),
    hair: String(loadout.hair),
    face: String(loadout.face),
    stance: opts.stance,
    frame: String(opts.frame),
    resize: String(opts.resize),
    items: items.join(','),
    gender: String(gender),
  });
  return `/api/assets/${tenant}/${region}/${major}.${minor}/character/${hash}.png?${params.toString()}`;
}

export function characterToLoadout(character: Character, inventory: Asset[]): CharacterLoadout {
  const equipment: Record<string, number> = {};
  for (const asset of inventory) {
    const slot = asset.attributes.slot;
    if (slot < 0) {
      equipment[String(slot)] = asset.attributes.templateId;
    }
  }
  return {
    skin: character.attributes.skinColor,
    hair: character.attributes.hair,
    face: character.attributes.face,
    equipment,
    gender: character.attributes.gender,
  };
}
