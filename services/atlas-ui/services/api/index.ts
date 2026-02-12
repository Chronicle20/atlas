/**
 * Service exports
 * Central export point for all API services
 */

// Export base service class
export { BaseService } from './base.service';

// Export types
export type {
  ServiceOptions,
  BatchOptions,
  QueryOptions,
  BatchResult,
  ValidationError,
} from './base.service';

// Export service-specific types
export type {
  TenantBasic,
  TenantBasicAttributes,
  TenantConfig,
  TenantConfigAttributes,
  Tenant,
  TenantAttributes,
} from './tenants.service';

export type {
  Account,
  AccountAttributes,
  AccountQueryOptions,
} from './accounts.service';

export type {
  Character,
  UpdateCharacterData,
} from '@/types/models/character';

export type {
  Inventory,
  Compartment,
  Asset,
  InventoryResponse,
  CompartmentType,
} from './inventory.service';

export type {
  Map,
  MapData,
  MapAttributes,
} from './maps.service';

export type {
  Guild,
  GuildAttributes,
  GuildMember,
} from './guilds.service';

export type {
  NPC,
  Shop,
  Commodity,
  CommodityAttributes,
  ShopResponse,
} from './npcs.service';

export type {
  ConversationCreateRequest,
  ConversationUpdateRequest,
  ConversationResponse,
  ConversationsResponse,
} from './conversations.service';

export type {
  TemplateCreateRequest,
  TemplateUpdateRequest,
  TemplateResponse,
  TemplatesResponse,
  TemplateOption,
} from './templates.service';

export type {
  CharacterRenderOptions,
  EquipmentData,
  MapleStoryCharacterData,
  CharacterImageResult,
  EquipmentExtractionResult,
  EquipmentExtractionOptions,
  CharacterRenderingConfig,
} from '@/types/models/maplestory';

// Individual services will be exported here as they are implemented:
export { tenantsService } from './tenants.service';
export { accountsService } from './accounts.service';
export { charactersService } from './characters.service';
export { inventoryService } from './inventory.service';
export { mapsService } from './maps.service';
export { guildsService } from './guilds.service';
export { npcsService } from './npcs.service';
export { conversationsService } from './conversations.service';
export { templatesService } from './templates.service';
export { MapleStoryService, mapleStoryService, mapSkinColor, getEquipmentSlotName, isTwoHandedWeapon } from './maplestory.service';
export { onboardingService, TemplateNotFoundError, TenantCreationError, ConfigurationCreationError } from './onboarding.service';
export type { OnboardResult } from './onboarding.service';

// Quest services
export { questsService } from './quests.service';
export type { QuestQueryOptions, QuestDefinition, QuestAttributes } from './quests.service';
export { questStatusService } from './quest-status.service';
export type { QuestStatusQueryOptions, CharacterQuestStatus, QuestState } from './quest-status.service';

// Service configuration
export { servicesService } from './services.service';
export type {
  Service,
  ServiceType,
  ServiceAttributes,
  LoginService,
  ChannelService,
  DropsService,
  LoginServiceAttributes,
  ChannelServiceAttributes,
  DropsServiceAttributes,
  CreateServiceInput,
  UpdateServiceInput,
  TaskConfig,
  LoginTenant,
  ChannelTenant,
} from './services.service';
export {
  isLoginService,
  isChannelService,
  isDropsService,
  getServiceTypeDisplayName,
  getServiceTenantCount,
  getServiceTaskCount,
  TASK_TYPES_BY_SERVICE,
} from './services.service';

// Game data services
export { monstersService } from './monsters.service';
export { reactorsService } from './reactors.service';
export { dropsService } from './drops.service';
export { gachaponsService } from './gachapons.service';
export { itemStringsService } from './item-strings.service';
export { itemsService } from './items.service';
export { portalScriptsService } from './portal-scripts.service';
export { reactorScriptsService } from './reactor-scripts.service';
export { seedService } from './seed.service';

export type { MonsterData, MonsterAttributes } from '@/types/models/monster';
export type { ReactorData, ReactorAttributes } from '@/types/models/reactor';
export type { DropData, ReactorDropData } from '@/types/models/drop';
export type { GachaponData, GachaponAttributes } from '@/types/models/gachapon';
export type { GachaponRewardData, GachaponRewardAttributes } from '@/types/models/gachapon-reward';
export type { ItemStringData, ItemStringAttributes } from '@/types/models/item-string';
export type { PortalScriptData } from './portal-scripts.service';
export type { ReactorScriptData } from './reactor-scripts.service';
export type { SeedResult } from './seed.service';
export { mapEntitiesService } from './map-entities.service';
export type { MapPortalData, MapNpcData, MapReactorData, MapMonsterData } from './map-entities.service';