/**
 * Template management service
 * Handles all template-related API operations
 */
import { BaseService } from './base.service';
import type { Template } from '@/types/models/template';

class TemplatesService extends BaseService {
  protected basePath = '/templates';

  /**
   * Get all templates
   */
  async getAll(signal?: AbortSignal): Promise<Template[]> {
    return super.getAll<Template>(signal);
  }

  /**
   * Get template by ID
   */
  async getById(id: string, signal?: AbortSignal): Promise<Template> {
    return super.getById<Template>(id, signal);
  }

  /**
   * Create new template
   */
  async create(data: Partial<Template>, signal?: AbortSignal): Promise<Template> {
    return super.create<Template, Partial<Template>>(data, signal);
  }

  /**
   * Update existing template
   */
  async update(id: string, data: Partial<Template>, signal?: AbortSignal): Promise<Template> {
    return super.update<Template, Partial<Template>>(id, data, signal);
  }

  /**
   * Delete template
   */
  async delete(id: string, signal?: AbortSignal): Promise<void> {
    return super.delete(id, signal);
  }
}

export const templatesService = new TemplatesService();