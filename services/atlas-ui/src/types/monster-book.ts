/**
 * Type definitions for the atlas-monster-book wire format.
 *
 * The backend (services/atlas-monster-book) emits JSON:API documents whose
 * resource `id` carries the natural key (characterId for the collection,
 * cardId for cards) and whose attributes mirror the fields below. The flat
 * shapes here are what callers see after the service's JSON:API → flat
 * transform.
 */

export interface MonsterBookCollection {
  characterId: number;
  bookLevel: number;
  normalCount: number;
  specialCount: number;
  totalUniqueCards: number;
  coverCardId: number;
  expBonusPercent: number;
}

export interface MonsterBookCard {
  cardId: number;
  level: number;
  isSpecial: boolean;
  firstAcquiredAt: string;
}

/** Wire-level JSON:API attributes (data.attributes) for the collection. */
export interface MonsterBookCollectionAttributes {
  bookLevel: number;
  normalCount: number;
  specialCount: number;
  totalUniqueCards: number;
  coverCardId: number;
  expBonusPercent: number;
}

/** Wire-level JSON:API attributes for a single card. */
export interface MonsterBookCardAttributes {
  level: number;
  isSpecial: boolean;
  firstAcquiredAt: string;
}
