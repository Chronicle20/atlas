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

// Individual services will be exported here as they are implemented:
// export { tenantsService } from './tenants.service';
// export { accountsService } from './accounts.service';
// export { charactersService } from './characters.service';
// export { inventoryService } from './inventory.service';
// export { mapsService } from './maps.service';
// export { guildsService } from './guilds.service';
// export { npcsService } from './npcs.service';
// export { conversationsService } from './conversations.service';
// export { templatesService } from './templates.service';