import { api } from "@/lib/api/client";
import {
  buildQueryString,
  runBatch,
  type QueryOptions,
  type ServiceOptions,
  type BatchOptions,
  type BatchResult,
  type ValidationError,
} from "@/lib/api/query-params";
import {
  fetchAll,
  fetchPaged,
  type PagedResult,
} from "@/services/api/pagination";
import type { Template, TemplateAttributes } from "@/types/models/template";

const BASE_PATH = "/api/configurations/templates";

export interface TemplateOption {
  id: string;
  attributes: {
    region: string;
    majorVersion: number;
    minorVersion: number;
  };
}

export interface TemplateCreateRequest {
  data: { type: "templates"; attributes: TemplateAttributes };
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
 * Normalises the collections atlas-configurations serialises as `null`.
 *
 * templates.RestModel declares `NPCs []npcs.RestModel` / `Worlds
 * []worlds.RestModel` without omitempty (templates/rest.go), so a template
 * whose stored document has no `npcs` key - five of the eleven seeds,
 * including template_jms_185_1.json - comes back as `"npcs": null`. That value
 * then fails validateTemplate's Array.isArray check on the way back out, which
 * broke every read-modify-write on those templates (the Packet Matrix's
 * mark-unsupported / add / delete flows all round-trip the whole document).
 * Absent and empty mean the same thing here, so read them as empty.
 */
function sortTemplate(template: Template): Template {
  if (!template.attributes.socket) return template;
  return {
    ...template,
    attributes: {
      ...template.attributes,
      npcs: template.attributes.npcs ?? [],
      worlds: template.attributes.worlds ?? [],
      socket: {
        ...template.attributes.socket,
        handlers: [...template.attributes.socket.handlers].sort(
          (a, b) => parseInt(a.opCode, 16) - parseInt(b.opCode, 16),
        ),
        writers: [...template.attributes.socket.writers].sort(
          (a, b) => parseInt(a.opCode, 16) - parseInt(b.opCode, 16),
        ),
      },
    },
  };
}

function compareTemplates(a: Template, b: Template): number {
  if (a.attributes.region !== b.attributes.region) {
    return a.attributes.region.localeCompare(b.attributes.region);
  }
  if (a.attributes.majorVersion !== b.attributes.majorVersion) {
    return a.attributes.majorVersion - b.attributes.majorVersion;
  }
  return a.attributes.minorVersion - b.attributes.minorVersion;
}

function sortAndTransform(templates: Template[]): Template[] {
  return templates.map(sortTemplate).sort(compareTemplates);
}

function validateTemplate(data: unknown): ValidationError[] {
  const errors: ValidationError[] = [];

  if (!data || typeof data !== "object") {
    errors.push({ field: "root", message: "Template data is required" });
    return errors;
  }

  const template = data as Partial<TemplateAttributes>;

  if (typeof template.region !== "string" || template.region.trim() === "") {
    errors.push({
      field: "region",
      message: "Region is required and must be a non-empty string",
      value: template.region,
    });
  }
  if (typeof template.majorVersion !== "number" || template.majorVersion < 0) {
    errors.push({
      field: "majorVersion",
      message: "Major version must be a non-negative number",
      value: template.majorVersion,
    });
  }
  if (typeof template.minorVersion !== "number" || template.minorVersion < 0) {
    errors.push({
      field: "minorVersion",
      message: "Minor version must be a non-negative number",
      value: template.minorVersion,
    });
  }
  if (typeof template.usesPin !== "boolean") {
    errors.push({
      field: "usesPin",
      message: "Uses pin must be a boolean value",
      value: template.usesPin,
    });
  }

  if (!template.characters || typeof template.characters !== "object") {
    errors.push({
      field: "characters",
      message: "Characters object is required",
      value: template.characters,
    });
  } else if (!Array.isArray(template.characters.templates)) {
    errors.push({
      field: "characters.templates",
      message: "Characters templates must be an array",
      value: template.characters.templates,
    });
  }

  if (!Array.isArray(template.npcs)) {
    errors.push({
      field: "npcs",
      message: "NPCs must be an array",
      value: template.npcs,
    });
  }

  if (!template.socket || typeof template.socket !== "object") {
    errors.push({
      field: "socket",
      message: "Socket object is required",
      value: template.socket,
    });
  } else {
    if (!Array.isArray(template.socket.handlers)) {
      errors.push({
        field: "socket.handlers",
        message: "Socket handlers must be an array",
        value: template.socket.handlers,
      });
    }
    if (!Array.isArray(template.socket.writers)) {
      errors.push({
        field: "socket.writers",
        message: "Socket writers must be an array",
        value: template.socket.writers,
      });
    }
  }

  if (!Array.isArray(template.worlds)) {
    errors.push({
      field: "worlds",
      message: "Worlds must be an array",
      value: template.worlds,
    });
  }

  return errors;
}

function throwIfInvalid(data: unknown, shouldValidate: boolean): void {
  if (!shouldValidate) return;
  const errors = validateTemplate(data);
  if (errors.length > 0) {
    throw new Error(
      `Template validation failed: ${errors.map((e) => e.message).join(", ")}`,
    );
  }
}

function wrapTemplate(
  attributes: TemplateAttributes,
  id?: string,
): TemplateCreateRequest | TemplateUpdateRequest {
  return {
    data: { type: "templates" as const, attributes, ...(id ? { id } : {}) },
  } as TemplateCreateRequest | TemplateUpdateRequest;
}

export const templatesService = {
  /**
   * Get every template, draining all pages (task-117). Used by consumers
   * that genuinely need the whole collection (region/version filtering,
   * export, consistency checks).
   */
  async getAll(options?: QueryOptions): Promise<Template[]> {
    const templates = await fetchAll<Template>(
      `${BASE_PATH}${buildQueryString(options)}`,
      undefined,
      options,
    );
    return sortAndTransform(templates);
  },

  /**
   * Get a single page of templates. Used by the Templates list view
   * (task-117), which pages server-side.
   */
  async getPage(
    page: { number: number; size: number },
    options?: QueryOptions,
  ): Promise<PagedResult<Template>> {
    const result = await fetchPaged<Template>(
      `${BASE_PATH}${buildQueryString(options)}`,
      page,
      options,
    );
    return { data: sortAndTransform(result.data), meta: result.meta };
  },

  async getById(id: string, options?: ServiceOptions): Promise<Template> {
    const template = await api.getOne<Template>(`${BASE_PATH}/${id}`, options);
    return sortTemplate(template);
  },

  /**
   * Get every template's region/version option, draining all pages
   * (task-117) — the dropdown this feeds needs the full set.
   */
  async getTemplateOptions(): Promise<TemplateOption[]> {
    const url = `${BASE_PATH}?fields[templates]=region,majorVersion,minorVersion`;
    const response = await fetchAll<TemplateOption>(url);
    return response.sort((a, b) => {
      if (a.attributes.region !== b.attributes.region)
        return a.attributes.region.localeCompare(b.attributes.region);
      if (a.attributes.majorVersion !== b.attributes.majorVersion)
        return a.attributes.majorVersion - b.attributes.majorVersion;
      return a.attributes.minorVersion - b.attributes.minorVersion;
    });
  },

  /**
   * Sparse read for the Packet Matrix: eleven templates at full attributes
   * carry character templates, presets and equipment lists the matrix never
   * reads.
   *
   * The result is READ-ONLY. It must never reach templatesService.update - a
   * sparse document in the write path erases characters/worlds/cashShop. It
   * lives under its own query key (socketKeys.matrix) for exactly that reason.
   */
  async getSocketMatrix(): Promise<Template[]> {
    const url = `${BASE_PATH}?fields[templates]=region,majorVersion,minorVersion,socket`;
    return fetchAll<Template>(url);
  },

  async exists(id: string, options?: ServiceOptions): Promise<boolean> {
    try {
      await templatesService.getById(id, options);
      return true;
    } catch (error) {
      if (
        error &&
        typeof error === "object" &&
        "status" in error &&
        (error as { status: number }).status === 404
      )
        return false;
      throw error;
    }
  },

  async create(
    data: TemplateAttributes,
    options?: ServiceOptions,
  ): Promise<Template> {
    throwIfInvalid(data, options?.validate !== false);
    const response = await api.post<TemplateResponse>(
      BASE_PATH,
      wrapTemplate(data),
      options,
    );
    return sortTemplate(response.data);
  },

  /**
   * Updates a template.
   *
   * `data` MUST be the WHOLE attribute document, not a partial: the request
   * body replaces the stored configuration wholesale. Passing a sparsely
   * fetched or partial object erases characters/worlds/cashShop, which is why
   * throwIfInvalid runs first and rejects anything missing a required field.
   *
   * Transport is PATCH. atlas-configurations binds /configurations/templates/{id}
   * to http.MethodPatch only (templates/resource.go) - there is no PUT route, so
   * the previous api.put call could only ever have 405'd.
   *
   * That PATCH handler writes NO response body on success
   * (handleUpdateConfigurationTemplate returns without marshalling anything), so
   * api.patch resolves to undefined and `response.data` threw "Cannot read
   * properties of undefined (reading 'data')". Fall back to the document we
   * just sent, the same way tenantsService.updateTenantConfiguration does.
   * Server-assigned values (character preset ids) still require a follow-up
   * GET - this return value is the request echo, not a server read.
   */
  async update(
    id: string,
    data: TemplateAttributes,
    options?: ServiceOptions,
  ): Promise<Template> {
    throwIfInvalid(data, options?.validate !== false);
    const response = await api.patch<TemplateResponse | undefined>(
      `${BASE_PATH}/${id}`,
      wrapTemplate(data, id),
      options,
    );
    return sortTemplate(response?.data ?? { id, attributes: data });
  },

  async patch(
    id: string,
    data: Partial<TemplateAttributes>,
    options?: ServiceOptions,
  ): Promise<Template> {
    const response = await api.patch<TemplateResponse | undefined>(
      `${BASE_PATH}/${id}`,
      wrapTemplate(data as TemplateAttributes, id),
      options,
    );
    return sortTemplate(
      response?.data ?? { id, attributes: data as TemplateAttributes },
    );
  },

  async delete(id: string, options?: ServiceOptions): Promise<void> {
    return api.delete(`${BASE_PATH}/${id}`, options);
  },

  /**
   * Resets one template to the configuration shipped in the currently deployed
   * image. Destructive: any edit made through the UI is overwritten.
   *
   * The endpoint returns 204 with no body, so there is nothing to sort or
   * validate on the way back out - the caller invalidates and refetches.
   *
   * Templates are global (NFR-5), so this request must NOT carry tenant
   * headers - unlike every other method on this service. `skipTenantHeaders`
   * always wins here regardless of what the caller passed, because the
   * endpoint is deliberately not tenant-scoped and TenantProvider sets a
   * tenant globally on the shared ApiClient singleton.
   */
  async reseed(id: string, options?: ServiceOptions): Promise<void> {
    await api.post<void>(`${BASE_PATH}/${id}/reseed`, undefined, {
      ...options,
      skipTenantHeaders: true,
    });
  },

  cloneTemplate(template: Template): TemplateAttributes {
    const cloned: TemplateAttributes = JSON.parse(
      JSON.stringify(template.attributes),
    );
    // shippedRevision/storedRevision/seedDrift are computed on read by
    // atlas-configurations, never persisted - cloning them would POST stale
    // values for a template that doesn't exist yet. Matches the same three
    // keys config-export.ts strips for the same reason.
    delete (cloned as Partial<TemplateAttributes>).shippedRevision;
    delete (cloned as Partial<TemplateAttributes>).storedRevision;
    delete (cloned as Partial<TemplateAttributes>).seedDrift;
    cloned.region = "";
    cloned.majorVersion = 0;
    cloned.minorVersion = 0;
    return cloned;
  },

  async createBatch(
    items: TemplateAttributes[],
    options?: ServiceOptions,
    batchOptions?: BatchOptions,
  ): Promise<BatchResult<Template>> {
    return runBatch(
      items,
      (item) => templatesService.create(item, options),
      batchOptions,
    );
  },

  async updateBatch(
    updates: Array<{ id: string; data: TemplateAttributes }>,
    options?: ServiceOptions,
    batchOptions?: BatchOptions,
  ): Promise<BatchResult<Template>> {
    return runBatch(
      updates,
      ({ id, data }) => templatesService.update(id, data, options),
      batchOptions,
    );
  },

  async deleteBatch(
    ids: string[],
    options?: ServiceOptions,
    batchOptions?: BatchOptions,
  ): Promise<BatchResult<string>> {
    return runBatch(
      ids,
      async (id) => {
        await templatesService.delete(id, options);
        return id;
      },
      batchOptions,
    );
  },

  async getByRegion(
    region: string,
    options?: QueryOptions,
  ): Promise<Template[]> {
    return templatesService.getAll({
      ...options,
      filters: { ...options?.filters, region },
    });
  },

  async getByVersion(
    majorVersion: number,
    minorVersion?: number,
    options?: QueryOptions,
  ): Promise<Template[]> {
    const filters: Record<string, unknown> = {
      ...options?.filters,
      majorVersion,
    };
    if (minorVersion !== undefined) filters.minorVersion = minorVersion;
    return templatesService.getAll({ ...options, filters });
  },

  async getByRegionAndVersion(
    region: string,
    majorVersion: number,
    minorVersion?: number,
    options?: ServiceOptions,
  ): Promise<Template[]> {
    const params = new URLSearchParams();
    params.append("region", region);
    params.append("majorVersion", majorVersion.toString());
    if (minorVersion !== undefined)
      params.append("minorVersion", minorVersion.toString());

    const response = await api.getOne<Template>(
      `${BASE_PATH}?${params.toString()}`,
      options,
    );
    return [sortTemplate(response)];
  },

  async export(
    format: "json" | "csv" = "json",
    options?: QueryOptions,
  ): Promise<Blob> {
    const templates = await templatesService.getAll(options);

    if (format === "csv") {
      const headers = [
        "ID",
        "Region",
        "Major Version",
        "Minor Version",
        "Uses Pin",
        "Character Templates Count",
        "NPCs Count",
        "Handlers Count",
        "Writers Count",
        "Worlds Count",
      ];
      const rows = templates.map((template) => [
        template.id,
        template.attributes.region,
        template.attributes.majorVersion.toString(),
        template.attributes.minorVersion.toString(),
        template.attributes.usesPin.toString(),
        template.attributes.characters.templates.length.toString(),
        template.attributes.npcs.length.toString(),
        template.attributes.socket.handlers.length.toString(),
        template.attributes.socket.writers.length.toString(),
        template.attributes.worlds.length.toString(),
      ]);
      const content = [headers, ...rows].map((row) => row.join(",")).join("\n");
      return new Blob([content], { type: "text/csv" });
    }

    return new Blob([JSON.stringify(templates, null, 2)], {
      type: "application/json",
    });
  },

  async validateTemplateConsistency(
    templateId: string,
  ): Promise<{ isValid: boolean; errors: string[] }> {
    const template = await templatesService.getById(templateId);
    const errors: string[] = [];

    template.attributes.characters.templates.forEach((charTemplate, index) => {
      if (charTemplate.faces.length === 0)
        errors.push(`Character template ${index}: No faces defined`);
      if (charTemplate.hairs.length === 0)
        errors.push(`Character template ${index}: No hairs defined`);
      if (charTemplate.hairColors.length === 0)
        errors.push(`Character template ${index}: No hair colors defined`);
      if (charTemplate.skinColors.length === 0)
        errors.push(`Character template ${index}: No skin colors defined`);
    });

    const npcIds = template.attributes.npcs.map((npc) => npc.npcId);
    const duplicateNpcIds = npcIds.filter(
      (id, index) => npcIds.indexOf(id) !== index,
    );
    if (duplicateNpcIds.length > 0)
      errors.push(`Duplicate NPC IDs found: ${duplicateNpcIds.join(", ")}`);

    const handlerOpCodes = template.attributes.socket.handlers.map(
      (h) => h.opCode,
    );
    const duplicateHandlerOpCodes = handlerOpCodes.filter(
      (op, index) => handlerOpCodes.indexOf(op) !== index,
    );
    if (duplicateHandlerOpCodes.length > 0)
      errors.push(
        `Duplicate handler opCodes found: ${duplicateHandlerOpCodes.join(", ")}`,
      );

    const writerOpCodes = template.attributes.socket.writers.map(
      (w) => w.opCode,
    );
    const duplicateWriterOpCodes = writerOpCodes.filter(
      (op, index) => writerOpCodes.indexOf(op) !== index,
    );
    if (duplicateWriterOpCodes.length > 0)
      errors.push(
        `Duplicate writer opCodes found: ${duplicateWriterOpCodes.join(", ")}`,
      );

    const worldNames = template.attributes.worlds.map((w) => w.name);
    const duplicateWorldNames = worldNames.filter(
      (name, index) => worldNames.indexOf(name) !== index,
    );
    if (duplicateWorldNames.length > 0)
      errors.push(
        `Duplicate world names found: ${duplicateWorldNames.join(", ")}`,
      );

    return { isValid: errors.length === 0, errors };
  },
};
