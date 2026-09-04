import { api } from "@/lib/api/client";
import {
  buildQueryString,
  type ServiceOptions,
  type QueryOptions,
} from "@/lib/api/query-params";
import type { ApiSingleResponse } from "@/types/api/responses";
import type { SocketConfig } from "@/types/models/socket";
import type { MapleLifeConfig } from "@/types/models/template";

const BASIC_PATH = "/api/tenants";
const CONFIG_PATH = "/api/configurations/tenants";

interface TenantBasicAttributes {
  name: string;
  region: string;
  majorVersion: number;
  minorVersion: number;
}

interface TenantBasic {
  id: string;
  attributes: TenantBasicAttributes;
}

interface CharacterPresetStatBlock {
  str: number;
  dex: number;
  int: number;
  luk: number;
  hp: number;
  mp: number;
}

interface CharacterPresetEquipmentEntry {
  templateId: number;
  useAverageStats: boolean;
}

interface CharacterPresetInventoryEntry {
  templateId: number;
  quantity: number;
}

interface CharacterPresetSkillEntry {
  skillId: number;
  level: number;
}

interface CharacterPresetAttributes {
  name: string;
  description: string;
  tags: string[];
  jobId: number;
  gender: 0 | 1;
  face: number;
  hair: number;
  hairColor: number;
  skinColor: number;
  mapId: number;
  level: number;
  meso: number;
  gm: number;
  stats: CharacterPresetStatBlock;
  defaultName: string;
  equipment: CharacterPresetEquipmentEntry[];
  inventory: CharacterPresetInventoryEntry[];
  skills: CharacterPresetSkillEntry[];
}

interface CharacterPreset {
  id?: string;
  attributes: CharacterPresetAttributes;
}

/**
 * The section names the reset endpoint accepts. `properties` is the
 * residual section — every comparable top-level key not claimed by one of
 * the five named sections, which today is exactly `usesPin`. The server
 * rejects `usesPin` as a name: there is no alias.
 */
export type TenantResetSection =
  "properties" | "socket" | "characters" | "npcs" | "cashShop" | "mapleLife";

interface TenantConfigAttributes {
  region: string;
  majorVersion: number;
  minorVersion: number;
  usesPin: boolean;
  characters: {
    templates: {
      jobIndex: number;
      subJobIndex: number;
      gender: number;
      mapId: number;
      faces: number[];
      hairs: number[];
      hairColors: number[];
      skinColors: number[];
      tops: number[];
      bottoms: number[];
      shoes: number[];
      weapons: number[];
      items: number[];
      skills: number[];
    }[];
    presets: CharacterPreset[];
  };
  npcs: {
    npcId: number;
    impl: string;
  }[];
  socket: SocketConfig;
  worlds: {
    name: string;
    flag: string;
    serverMessage: string;
    eventMessage: string;
    whyAmIRecommended: string;
    expRate?: number;
    mesoRate?: number;
    itemDropRate?: number;
    questExpRate?: number;
  }[];
  cashShop?: {
    commodities: {
      hourlyExpirations?: {
        templateId: number;
        hours: number;
      }[];
    };
    /**
     * Cash item template ids that open as a Cash Shop Surprise box. An empty
     * or absent list is NOT "the feature is off" — atlas-cashshop falls back
     * to the stock box 5222000 (configuration/registry.go
     * GetSurpriseBoxTemplateIds), so readers must apply the same fallback.
     */
    surprise?: {
      boxTemplateIds?: number[];
    };
  };
  mapleLife?: MapleLifeConfig;
  diagnostics?: { tracePackets: boolean };
  /**
   * Computed server-side (task-289); never persisted and ignored on write.
   * Optional so a response from an older backend still type-checks — read
   * `templateDrift === true`, never a truthy check.
   */
  baselineTemplateId?: string;
  baselineRevision?: string;
  storedRevision?: string;
  templateDrift?: boolean;
  /** Always fully populated by a current backend: all six section keys. */
  sectionDrift?: Record<string, boolean>;
}

interface TenantConfig {
  id: string;
  attributes: TenantConfigAttributes;
}

export type TenantAttributes = TenantConfigAttributes;
export type Tenant = TenantBasic;

interface CreateTenantInput {
  data: { type: "tenants"; attributes: TenantBasicAttributes };
}

interface UpdateTenantInput {
  data: {
    id: string;
    type: "tenants";
    attributes: Partial<TenantBasicAttributes>;
  };
}

interface CreateTenantConfigInput {
  data: { id?: string; type: "tenants"; attributes: TenantConfigAttributes };
}

interface UpdateTenantConfigInput {
  data: {
    id: string;
    type: "tenants";
    attributes: Partial<TenantConfigAttributes>;
  };
}

function sortTenants<T extends TenantBasic | TenantConfig>(tenants: T[]): T[] {
  return tenants.sort((a, b) => {
    if (a.attributes.region !== b.attributes.region) {
      return a.attributes.region.localeCompare(b.attributes.region);
    }
    if (a.attributes.majorVersion !== b.attributes.majorVersion) {
      return a.attributes.majorVersion - b.attributes.majorVersion;
    }
    return a.attributes.minorVersion - b.attributes.minorVersion;
  });
}

function sortTenantConfig(config: TenantConfig): TenantConfig {
  if (!config.attributes.socket) return config;
  return {
    ...config,
    attributes: {
      ...config.attributes,
      socket: {
        ...config.attributes.socket,
        handlers: [...config.attributes.socket.handlers].sort(
          (a, b) => parseInt(a.opCode, 16) - parseInt(b.opCode, 16),
        ),
        writers: [...config.attributes.socket.writers].sort(
          (a, b) => parseInt(a.opCode, 16) - parseInt(b.opCode, 16),
        ),
      },
    },
  };
}

export const tenantsService = {
  async getAllTenants(options?: QueryOptions): Promise<TenantBasic[]> {
    const tenants = await api.getList<TenantBasic>(
      `${BASIC_PATH}${buildQueryString(options)}`,
      options,
    );
    return sortTenants(tenants);
  },

  async getTenantById(
    id: string,
    options?: ServiceOptions,
  ): Promise<TenantBasic> {
    return api.getOne<TenantBasic>(`${BASIC_PATH}/${id}`, options);
  },

  async createTenant(
    attributes: TenantBasicAttributes,
    options?: ServiceOptions,
  ): Promise<TenantBasic> {
    const input: CreateTenantInput = { data: { type: "tenants", attributes } };
    const response = await api.post<ApiSingleResponse<TenantBasic>>(
      BASIC_PATH,
      input,
      options,
    );
    return response.data;
  },

  async updateTenant(
    tenant: TenantBasic,
    updatedAttributes: Partial<TenantBasicAttributes>,
    options?: ServiceOptions,
  ): Promise<TenantBasic> {
    const input: UpdateTenantInput = {
      data: {
        id: tenant.id,
        type: "tenants",
        attributes: { ...tenant.attributes, ...updatedAttributes },
      },
    };
    await api.patch<void>(`${BASIC_PATH}/${tenant.id}`, input, options);
    return {
      ...tenant,
      attributes: { ...tenant.attributes, ...updatedAttributes },
    };
  },

  async deleteTenant(
    tenantId: string,
    options?: ServiceOptions,
  ): Promise<void> {
    return api.delete(`${BASIC_PATH}/${tenantId}`, options);
  },

  // Tenant configuration methods (separate endpoint under /api/configurations/tenants).

  async getAllTenantConfigurations(
    options?: QueryOptions,
  ): Promise<TenantConfig[]> {
    const configs = await api.getList<TenantConfig>(
      `${CONFIG_PATH}${buildQueryString(options)}`,
      options,
    );
    return sortTenants(configs).map(sortTenantConfig);
  },

  async getTenantConfigurationById(
    id: string,
    options?: ServiceOptions,
  ): Promise<TenantConfig> {
    const config = await api.getOne<TenantConfig>(
      `${CONFIG_PATH}/${id}`,
      options,
    );
    return sortTenantConfig(config);
  },

  /**
   * Sparse read of every tenant configuration, for the Packet Matrix's tenant
   * columns. READ-ONLY - see templatesService.getSocketMatrix. Not routed
   * through sortTenantConfig: that reorders handlers/writers for display,
   * which the matrix does on its own, and skipping it here means the sparse
   * response passes through untouched (including socket.unsupported).
   */
  async getSocketMatrix(options?: ServiceOptions): Promise<TenantConfig[]> {
    const url = `${CONFIG_PATH}?fields[tenants]=region,majorVersion,minorVersion,socket`;
    return api.getList<TenantConfig>(url, options);
  },

  async createTenantConfiguration(
    tenantId: string,
    attributes: TenantConfigAttributes,
    options?: ServiceOptions,
  ): Promise<TenantConfig> {
    const input: CreateTenantConfigInput = {
      data: { id: tenantId, type: "tenants", attributes },
    };
    const response = await api.post<ApiSingleResponse<TenantConfig>>(
      CONFIG_PATH,
      input,
      options,
    );
    return response.data;
  },

  async updateTenantConfiguration(
    tenant: TenantConfig,
    updatedAttributes: Partial<TenantConfigAttributes>,
    options?: ServiceOptions,
  ): Promise<TenantConfig> {
    // The five computed attributes are read-only and server-owned. The
    // server ignores them (they are absent from the bound write model),
    // so this is hygiene rather than a fix — it keeps request bodies
    // honest instead of echoing a hash of the document back at the
    // service that produced it.
    const {
      baselineTemplateId: _baselineTemplateId,
      baselineRevision: _baselineRevision,
      storedRevision: _storedRevision,
      templateDrift: _templateDrift,
      sectionDrift: _sectionDrift,
      ...writable
    } = tenant.attributes;

    const input: UpdateTenantConfigInput = {
      data: {
        id: tenant.id,
        type: "tenants",
        attributes: { ...writable, ...updatedAttributes },
      },
    };
    await api.patch<void>(`${CONFIG_PATH}/${tenant.id}`, input, options);
    return {
      ...tenant,
      attributes: { ...tenant.attributes, ...updatedAttributes },
    };
  },

  /**
   * Resets a tenant configuration to its baseline template. Omit
   * `sections` for the whole document.
   *
   * Unlike templatesService.reseed this does NOT set
   * `skipTenantHeaders` — templates are global, tenant configurations are
   * not, and the reset must carry the ordinary tenant headers.
   */
  async reset(
    id: string,
    sections?: TenantResetSection[],
    options?: ServiceOptions,
  ): Promise<TenantConfig> {
    const body =
      sections && sections.length > 0
        ? { data: { type: "tenants", attributes: { sections } } }
        : undefined;
    const response = await api.post<ApiSingleResponse<TenantConfig>>(
      `${CONFIG_PATH}/${id}/reset`,
      body,
      options,
    );
    return sortTenantConfig(response.data);
  },

  createTenantFromTemplate(template: {
    attributes: TenantConfigAttributes;
  }): TenantConfigAttributes {
    return JSON.parse(JSON.stringify(template.attributes));
  },
};

export type {
  TenantBasic,
  TenantBasicAttributes,
  TenantConfig,
  TenantConfigAttributes,
};
