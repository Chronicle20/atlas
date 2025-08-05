/**
 * MapleStory character rendering types and interfaces
 * Used for character visualization and equipment rendering via MapleStory.io API
 */

import type { Character } from './character';
import type { Asset } from '@/services/api/inventory.service';

/**
 * Weapon type enumeration based on MapleStory item classification
 * Matches the Go backend weapon type constants
 */
export enum WeaponType {
  OneHandedSword = 0,
  OneHandedAxe = 1,
  OneHandedMace = 2,
  Dagger = 3,
  Wand = 7,
  Staff = 8,
  TwoHandedSword = 10,
  TwoHandedAxe = 11,
  TwoHandedMace = 12,
  Spear = 13,
  Polearm = 14,
  Bow = 15,
  Crossbow = 16,
  Claw = 17,
  Knuckle = 18,
  Gun = 19,
  None = 99,
}

/**
 * Equipment data structure mapping inventory slots to item IDs
 * Negative slot indexes represent equipped items
 */
export interface EquipmentData {
  [slot: string]: number | undefined;
  // Standard equipment slots (-1 to -49)
  '-1'?: number;   // Hat
  '-5'?: number;   // Top/Overall
  '-6'?: number;   // Bottom
  '-7'?: number;   // Shoes
  '-8'?: number;   // Gloves
  '-9'?: number;   // Cape
  '-10'?: number;  // Shield
  '-11'?: number;  // Weapon
  '-12'?: number;  // Ring
  '-13'?: number;  // Ring
  '-14'?: number;  // Ring
  '-15'?: number;  // Ring
  '-16'?: number;  // Pendant
  '-17'?: number;  // Belt
  '-18'?: number;  // Medal
  '-19'?: number;  // Shoulder
  '-20'?: number;  // Pocket Item
  '-21'?: number;  // Eye Accessory
  '-22'?: number;  // Face Accessory
  '-23'?: number;  // Earrings
  '-24'?: number;  // Emblem
  '-25'?: number;  // Badge
  // Cash equipment slots (-101 to -114)
  '-101'?: number; // Cash Hat
  '-102'?: number; // Cash Face
  '-103'?: number; // Cash Eye
  '-104'?: number; // Cash Top
  '-105'?: number; // Cash Overall
  '-106'?: number; // Cash Bottom
  '-107'?: number; // Cash Shoes
  '-108'?: number; // Cash Gloves
  '-109'?: number; // Cash Cape
  '-110'?: number; // Cash Shield
  '-111'?: number; // Cash Weapon
  '-112'?: number; // Cash Ring
  '-113'?: number; // Cash Pendant
  '-114'?: number; // Cash Belt
}

/**
 * Character rendering options for MapleStory.io API
 */
export interface CharacterRenderOptions {
  hair: number;
  face: number;
  skin: number;
  equipment: EquipmentData;
  stance?: 'stand1' | 'stand2';
  frame?: number;
  resize?: number;
  renderMode?: 'default' | 'compact';
  flipX?: boolean;
}

/**
 * MapleStory character data extracted from Character model
 */
export interface MapleStoryCharacterData {
  id: string;
  name: string;
  level: number;
  jobId: number;
  hair: number;
  face: number;
  skinColor: number;
  gender: number;
  equipment: EquipmentData;
}

/**
 * Equipment slot configuration for rendering order and positioning
 */
export interface EquipmentSlotConfig {
  slot: string;
  name: string;
  category: 'weapon' | 'armor' | 'accessory' | 'cash';
  renderOrder: number;
  twoHanded?: boolean;
}

/**
 * Stance configuration based on weapon type
 */
export interface StanceConfig {
  weaponId: number;
  stance: 'stand1' | 'stand2';
  category: 'one-handed' | 'two-handed';
}

/**
 * Skin color mapping for MapleStory.io API
 */
export interface SkinColorMapping {
  [key: number]: number;
}

/**
 * Equipment slot name mapping
 */
export interface EquipmentSlotMapping {
  [slot: string]: string;
}

/**
 * Two-handed weapon ID ranges
 */
export interface WeaponRange {
  min: number;
  max: number;
  category: string;
}

/**
 * Character image generation result
 */
export interface CharacterImageResult {
  url: string;
  character: MapleStoryCharacterData;
  options: CharacterRenderOptions;
  cached: boolean;
}

/**
 * Equipment extraction result from inventory
 */
export interface EquipmentExtractionResult {
  equipment: EquipmentData;
  equippedCount: number;
  totalSlots: number;
}

/**
 * Character rendering component props
 */
export interface CharacterRendererProps {
  character: Character;
  equipment?: EquipmentData;
  scale?: number;
  showLoading?: boolean;
  fallbackAvatar?: string;
  className?: string;
  onImageLoad?: () => void;
  onImageError?: (error: Error) => void;
}

/**
 * Equipment extraction options
 */
export interface EquipmentExtractionOptions {
  includeNegativeSlots?: boolean;
  includeCashEquipment?: boolean;
  filterBySlotRange?: {
    min: number;
    max: number;
  };
}

/**
 * NPC API response data from MapleStory.io
 */
export interface NpcApiData {
  id: number;
  name: string;
  description?: string;
  scripts?: string[];
}

/**
 * NPC icon and name result
 */
export interface NpcDataResult {
  id: number;
  name?: string;
  iconUrl?: string;
  cached: boolean;
  error?: string;
}

/**
 * Character rendering service configuration
 */
export interface CharacterRenderingConfig {
  apiBaseUrl: string;
  apiVersion: string;
  cacheEnabled: boolean;
  cacheTTL: number;
  defaultStance: 'stand1' | 'stand2';
  defaultResize: number;
  enableErrorLogging: boolean;
  defaultRegion?: string;
}