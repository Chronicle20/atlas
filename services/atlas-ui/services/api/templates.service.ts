/**
 * Template service
 * Handles all template-related API operations with full API client feature support
 */
import { BaseService } from './base.service';
import type { 
  Template, 
  TemplateAttributes,
  CharacterTemplate
} from '@/types/models/template';

/**
 * Request/response interfaces for API communication
 */
export interface TemplateCreateRequest {
  data: {
    type: "templates";
    attributes: TemplateAttributes;
  };
}

export interface TemplateUpdateRequest {
  data: {
    type: "templates";
    id: string;
    attributes: Partial<TemplateAttributes>;
  };
}

export interface TemplateResponse {
  data: Template;
}

export interface TemplatesResponse {
  data: Template[];
}

/**
 * Template service class extending BaseService
 * Provides comprehensive template management with specialized sorting and validation
 */
class TemplatesService extends BaseService {
  protected basePath = '/api/configurations/templates';

  /**
   * Transform request data to API format
   */
  protected transformRequest<T>(data: T): T {
    // For create/update operations, wrap data in the expected format
    if (data && typeof data === 'object' && !('data' in data)) {
      return {
        data: {
          type: "templates",
          attributes: data,
          ...(('id' in data && data.id) ? { id: data.id } : {})
        }
      } as T;
    }
    return data;
  }

  /**
   * Transform API response to domain model format with sorting
   */
  protected transformResponse<T>(data: T): T {
    // Extract data from API response wrapper if present
    let extractedData = data;
    if (data && typeof data === 'object' && 'data' in data) {
      extractedData = (data as any).data;
    }

    // If it's a template or array of templates, apply sorting transformations
    if (Array.isArray(extractedData)) {
      return extractedData
        .map((template: Template) => this.sortTemplateHandlersAndWriters(template))
        .sort((a: Template, b: Template) => this.compareTemplates(a, b)) as T;
    } else if (extractedData && typeof extractedData === 'object' && 'attributes' in extractedData) {
      return this.sortTemplateHandlersAndWriters(extractedData as Template) as T;
    }

    return extractedData;
  }

  /**
   * Sort socket handlers and writers by opCode (hex values)
   */
  private sortTemplateHandlersAndWriters(template: Template): Template {
    return {
      ...template,
      attributes: {
        ...template.attributes,
        socket: {
          ...template.attributes.socket,
          handlers: [...template.attributes.socket.handlers].sort(
            (a, b) => parseInt(a.opCode, 16) - parseInt(b.opCode, 16)
          ),
          writers: [...template.attributes.socket.writers].sort(
            (a, b) => parseInt(a.opCode, 16) - parseInt(b.opCode, 16)
          ),
        },
      },
    };
  }

  /**
   * Compare templates for sorting (region, majorVersion, minorVersion)
   */
  private compareTemplates(a: Template, b: Template): number {
    if (a.attributes.region !== b.attributes.region) {
      return a.attributes.region.localeCompare(b.attributes.region);
    }
    if (a.attributes.majorVersion !== b.attributes.majorVersion) {
      return a.attributes.majorVersion - b.attributes.majorVersion;
    }
    return a.attributes.minorVersion - b.attributes.minorVersion;
  }

  /**
   * Validate template data before API calls
   */
  protected validate(data: unknown): Array<{ field: string; message: string; value?: unknown }> {
    const errors: Array<{ field: string; message: string; value?: unknown }> = [];
    
    if (!data || typeof data !== 'object') {
      errors.push({ field: 'root', message: 'Template data is required' });
      return errors;
    }

    const template = data as Partial<TemplateAttributes>;

    // Validate required fields
    if (typeof template.region !== 'string' || template.region.trim() === '') {
      errors.push({ field: 'region', message: 'Region is required and must be a non-empty string', value: template.region });
    }

    if (typeof template.majorVersion !== 'number' || template.majorVersion < 0) {
      errors.push({ field: 'majorVersion', message: 'Major version must be a non-negative number', value: template.majorVersion });
    }

    if (typeof template.minorVersion !== 'number' || template.minorVersion < 0) {
      errors.push({ field: 'minorVersion', message: 'Minor version must be a non-negative number', value: template.minorVersion });
    }

    if (typeof template.usesPin !== 'boolean') {
      errors.push({ field: 'usesPin', message: 'Uses pin must be a boolean value', value: template.usesPin });
    }

    // Validate characters structure
    if (!template.characters || typeof template.characters !== 'object') {
      errors.push({ field: 'characters', message: 'Characters object is required', value: template.characters });
    } else if (!Array.isArray(template.characters.templates)) {
      errors.push({ field: 'characters.templates', message: 'Characters templates must be an array', value: template.characters.templates });
    } else {
      // Validate each character template
      template.characters.templates.forEach((charTemplate, index) => {
        if (typeof charTemplate.jobIndex !== 'number' || charTemplate.jobIndex < 0) {
          errors.push({ field: `characters.templates[${index}].jobIndex`, message: 'Job index must be a non-negative number', value: charTemplate.jobIndex });
        }
        if (typeof charTemplate.subJobIndex !== 'number' || charTemplate.subJobIndex < 0) {
          errors.push({ field: `characters.templates[${index}].subJobIndex`, message: 'Sub job index must be a non-negative number', value: charTemplate.subJobIndex });
        }
        if (typeof charTemplate.gender !== 'number' || (charTemplate.gender !== 0 && charTemplate.gender !== 1)) {
          errors.push({ field: `characters.templates[${index}].gender`, message: 'Gender must be 0 or 1', value: charTemplate.gender });
        }
        if (typeof charTemplate.mapId !== 'number' || charTemplate.mapId < 0) {
          errors.push({ field: `characters.templates[${index}].mapId`, message: 'Map ID must be a non-negative number', value: charTemplate.mapId });
        }
        
        // Validate arrays
        const arrayFields = ['faces', 'hairs', 'hairColors', 'skinColors', 'tops', 'bottoms', 'shoes', 'weapons', 'items', 'skills'];
        arrayFields.forEach(field => {
          if (!Array.isArray((charTemplate as any)[field])) {
            errors.push({ field: `characters.templates[${index}].${field}`, message: `${field} must be an array`, value: (charTemplate as any)[field] });
          }
        });
      });
    }

    // Validate NPCs array
    if (!Array.isArray(template.npcs)) {
      errors.push({ field: 'npcs', message: 'NPCs must be an array', value: template.npcs });
    } else {
      template.npcs.forEach((npc, index) => {
        if (typeof npc.npcId !== 'number' || npc.npcId <= 0) {
          errors.push({ field: `npcs[${index}].npcId`, message: 'NPC ID must be a positive number', value: npc.npcId });
        }
        if (typeof npc.impl !== 'string' || npc.impl.trim() === '') {
          errors.push({ field: `npcs[${index}].impl`, message: 'NPC implementation must be a non-empty string', value: npc.impl });
        }
      });
    }

    // Validate socket structure
    if (!template.socket || typeof template.socket !== 'object') {
      errors.push({ field: 'socket', message: 'Socket object is required', value: template.socket });
    } else {
      if (!Array.isArray(template.socket.handlers)) {
        errors.push({ field: 'socket.handlers', message: 'Socket handlers must be an array', value: template.socket.handlers });
      } else {
        template.socket.handlers.forEach((handler, index) => {
          if (typeof handler.opCode !== 'string' || handler.opCode.trim() === '') {
            errors.push({ field: `socket.handlers[${index}].opCode`, message: 'Handler opCode must be a non-empty string', value: handler.opCode });
          }
          if (typeof handler.validator !== 'string' || handler.validator.trim() === '') {
            errors.push({ field: `socket.handlers[${index}].validator`, message: 'Handler validator must be a non-empty string', value: handler.validator });
          }
          if (typeof handler.handler !== 'string' || handler.handler.trim() === '') {
            errors.push({ field: `socket.handlers[${index}].handler`, message: 'Handler handler must be a non-empty string', value: handler.handler });
          }
        });
      }

      if (!Array.isArray(template.socket.writers)) {
        errors.push({ field: 'socket.writers', message: 'Socket writers must be an array', value: template.socket.writers });
      } else {
        template.socket.writers.forEach((writer, index) => {
          if (typeof writer.opCode !== 'string' || writer.opCode.trim() === '') {
            errors.push({ field: `socket.writers[${index}].opCode`, message: 'Writer opCode must be a non-empty string', value: writer.opCode });
          }
          if (typeof writer.writer !== 'string' || writer.writer.trim() === '') {
            errors.push({ field: `socket.writers[${index}].writer`, message: 'Writer writer must be a non-empty string', value: writer.writer });
          }
        });
      }
    }

    // Validate worlds array
    if (!Array.isArray(template.worlds)) {
      errors.push({ field: 'worlds', message: 'Worlds must be an array', value: template.worlds });
    } else {
      template.worlds.forEach((world, index) => {
        if (typeof world.name !== 'string' || world.name.trim() === '') {
          errors.push({ field: `worlds[${index}].name`, message: 'World name must be a non-empty string', value: world.name });
        }
        if (typeof world.flag !== 'string' || world.flag.trim() === '') {
          errors.push({ field: `worlds[${index}].flag`, message: 'World flag must be a non-empty string', value: world.flag });
        }
        if (typeof world.serverMessage !== 'string') {
          errors.push({ field: `worlds[${index}].serverMessage`, message: 'World server message must be a string', value: world.serverMessage });
        }
        if (typeof world.eventMessage !== 'string') {
          errors.push({ field: `worlds[${index}].eventMessage`, message: 'World event message must be a string', value: world.eventMessage });
        }
        if (typeof world.whyAmIRecommended !== 'string') {
          errors.push({ field: `worlds[${index}].whyAmIRecommended`, message: 'World why am I recommended must be a string', value: world.whyAmIRecommended });
        }
      });
    }

    return errors;
  }

  /**
   * Get all templates with advanced query support and automatic sorting
   */
  async getAll(options?: Parameters<BaseService['getAll']>[0]): Promise<Template[]> {
    return super.getAll<Template>(options);
  }

  /**
   * Get template by ID
   */
  async getById(id: string, options?: Parameters<BaseService['getById']>[1]): Promise<Template> {
    return super.getById<Template>(id, options);
  }

  /**
   * Check if template exists
   */
  async exists(id: string, options?: Parameters<BaseService['exists']>[1]): Promise<boolean> {
    return super.exists(id, options);
  }

  /**
   * Create new template with validation
   */
  async create(templateAttributes: TemplateAttributes, options?: Parameters<BaseService['create']>[1]): Promise<Template> {
    return super.create<Template, TemplateAttributes>(templateAttributes, { validate: true, ...options });
  }

  /**
   * Update existing template with validation
   */
  async update(id: string, templateAttributes: Partial<TemplateAttributes>, options?: Parameters<BaseService['update']>[2]): Promise<Template> {
    return super.update<Template, Partial<TemplateAttributes>>(id, templateAttributes, { validate: true, ...options });
  }

  /**
   * Partially update template (PATCH)
   */
  async patch(id: string, updates: Partial<TemplateAttributes>, options?: Parameters<BaseService['patch']>[2]): Promise<Template> {
    return super.patch<Template, Partial<TemplateAttributes>>(id, updates, options);
  }

  /**
   * Delete template
   */
  async delete(id: string, options?: Parameters<BaseService['delete']>[1]): Promise<void> {
    return super.delete(id, options);
  }

  /**
   * Clone template attributes for creating a new template
   * Clears region, majorVersion, and minorVersion as they should be unique
   */
  cloneTemplate(template: Template): TemplateAttributes {
    // Create a deep copy of the template attributes
    const clonedAttributes: TemplateAttributes = JSON.parse(JSON.stringify(template.attributes));

    // Clear the region, majorVersion, and minorVersion as required
    clonedAttributes.region = "";
    clonedAttributes.majorVersion = 0;
    clonedAttributes.minorVersion = 0;

    return clonedAttributes;
  }

  /**
   * Batch create multiple templates
   */
  async createBatch(
    templates: TemplateAttributes[], 
    options?: Parameters<BaseService['createBatch']>[1],
    batchOptions?: Parameters<BaseService['createBatch']>[2]
  ) {
    return super.createBatch<Template, TemplateAttributes>(
      templates, 
      { validate: true, ...options },
      batchOptions
    );
  }

  /**
   * Batch update multiple templates
   */
  async updateBatch(
    updates: Array<{ id: string; data: Partial<TemplateAttributes> }>,
    options?: Parameters<BaseService['updateBatch']>[1],
    batchOptions?: Parameters<BaseService['updateBatch']>[2]
  ) {
    return super.updateBatch<Template, Partial<TemplateAttributes>>(
      updates,
      { validate: true, ...options },
      batchOptions
    );
  }

  /**
   * Batch delete multiple templates
   */
  async deleteBatch(
    ids: string[],
    options?: Parameters<BaseService['deleteBatch']>[1],
    batchOptions?: Parameters<BaseService['deleteBatch']>[2]
  ) {
    return super.deleteBatch(ids, options, batchOptions);
  }

  /**
   * Search templates by region
   */
  async getByRegion(region: string, options?: Parameters<BaseService['getAll']>[0]): Promise<Template[]> {
    return this.getAll({
      ...options,
      filters: {
        ...options?.filters,
        region: region
      }
    });
  }

  /**
   * Search templates by version
   */
  async getByVersion(majorVersion: number, minorVersion?: number, options?: Parameters<BaseService['getAll']>[0]): Promise<Template[]> {
    const filters: Record<string, unknown> = {
      ...options?.filters,
      majorVersion: majorVersion
    };

    if (minorVersion !== undefined) {
      filters.minorVersion = minorVersion;
    }

    return this.getAll({
      ...options,
      filters
    });
  }

  /**
   * Get templates by region and version combination
   */
  async getByRegionAndVersion(
    region: string, 
    majorVersion: number, 
    minorVersion?: number, 
    options?: Parameters<BaseService['getAll']>[0]
  ): Promise<Template[]> {
    const filters: Record<string, unknown> = {
      ...options?.filters,
      region: region,
      majorVersion: majorVersion
    };

    if (minorVersion !== undefined) {
      filters.minorVersion = minorVersion;
    }

    return this.getAll({
      ...options,
      filters
    });
  }

  /**
   * Export template data (useful for backups or migrations)  
   */
  async export(format: 'json' | 'csv' = 'json', options?: Parameters<BaseService['getAll']>[0]): Promise<Blob> {
    const templates = await this.getAll(options);
    
    let content: string;
    let mimeType: string;
    
    if (format === 'csv') {
      // Convert to CSV format
      const headers = ['ID', 'Region', 'Major Version', 'Minor Version', 'Uses Pin', 'Character Templates Count', 'NPCs Count', 'Handlers Count', 'Writers Count', 'Worlds Count'];
      const rows = templates.map(template => [
        template.id,
        template.attributes.region,
        template.attributes.majorVersion.toString(),
        template.attributes.minorVersion.toString(),
        template.attributes.usesPin.toString(),
        template.attributes.characters.templates.length.toString(),
        template.attributes.npcs.length.toString(),
        template.attributes.socket.handlers.length.toString(),
        template.attributes.socket.writers.length.toString(),
        template.attributes.worlds.length.toString()
      ]);
      
      content = [headers, ...rows].map(row => row.join(',')).join('\n');
      mimeType = 'text/csv';
    } else {
      content = JSON.stringify(templates, null, 2);
      mimeType = 'application/json';
    }
    
    return new Blob([content], { type: mimeType });
  }

  /**
   * Validate template consistency
   * Checks that all references and configurations are valid
   */
  async validateTemplateConsistency(templateId: string): Promise<{ isValid: boolean; errors: string[] }> {
    const template = await this.getById(templateId);
    const errors: string[] = [];
    
    // Check character templates
    template.attributes.characters.templates.forEach((charTemplate, index) => {
      if (charTemplate.faces.length === 0) {
        errors.push(`Character template ${index}: No faces defined`);
      }
      if (charTemplate.hairs.length === 0) {
        errors.push(`Character template ${index}: No hairs defined`);
      }
      if (charTemplate.hairColors.length === 0) {
        errors.push(`Character template ${index}: No hair colors defined`);
      }
      if (charTemplate.skinColors.length === 0) {
        errors.push(`Character template ${index}: No skin colors defined`);
      }
    });
    
    // Check for duplicate NPC IDs
    const npcIds = template.attributes.npcs.map(npc => npc.npcId);
    const duplicateNpcIds = npcIds.filter((id, index) => npcIds.indexOf(id) !== index);
    if (duplicateNpcIds.length > 0) {
      errors.push(`Duplicate NPC IDs found: ${duplicateNpcIds.join(', ')}`);
    }
    
    // Check for duplicate socket handler opCodes
    const handlerOpCodes = template.attributes.socket.handlers.map(handler => handler.opCode);
    const duplicateHandlerOpCodes = handlerOpCodes.filter((opCode, index) => handlerOpCodes.indexOf(opCode) !== index);
    if (duplicateHandlerOpCodes.length > 0) {
      errors.push(`Duplicate handler opCodes found: ${duplicateHandlerOpCodes.join(', ')}`);
    }
    
    // Check for duplicate socket writer opCodes
    const writerOpCodes = template.attributes.socket.writers.map(writer => writer.opCode);
    const duplicateWriterOpCodes = writerOpCodes.filter((opCode, index) => writerOpCodes.indexOf(opCode) !== index);
    if (duplicateWriterOpCodes.length > 0) {
      errors.push(`Duplicate writer opCodes found: ${duplicateWriterOpCodes.join(', ')}`);
    }
    
    // Check for duplicate world names
    const worldNames = template.attributes.worlds.map(world => world.name);
    const duplicateWorldNames = worldNames.filter((name, index) => worldNames.indexOf(name) !== index);
    if (duplicateWorldNames.length > 0) {
      errors.push(`Duplicate world names found: ${duplicateWorldNames.join(', ')}`);
    }
    
    return {
      isValid: errors.length === 0,
      errors
    };
  }

  /**
   * Clear templates cache
   */
  clearCache(): void {
    super.clearServiceCache();
  }

  /**
   * Get cache statistics for templates
   */
  getCacheStats() {
    return super.getServiceCacheStats();
  }
}

// Export singleton instance
export const templatesService = new TemplatesService();