import { api } from "@/lib/api/client";
import { buildQueryString, type ServiceOptions, type QueryOptions } from "@/lib/api/query-params";
import type { ApiSingleResponse } from "@/types/api/responses";

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
  };
  npcs: {
    npcId: number;
    impl: string;
  }[];
  socket: {
    handlers: {
      opCode: string;
      validator: string;
      handler: string;
      options: unknown;
    }[];
    writers: {
      opCode: string;
      writer: string;
      options: unknown;
    }[];
  };
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
  };
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
  data: { id: string; type: "tenants"; attributes: Partial<TenantBasicAttributes> };
}

interface CreateTenantConfigInput {
  data: { id?: string; type: "tenants"; attributes: TenantConfigAttributes };
}

interface UpdateTenantConfigInput {
  data: { id: string; type: "tenants"; attributes: Partial<TenantConfigAttributes> };
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
    const tenants = await api.getList<TenantBasic>(`${BASIC_PATH}${buildQueryString(options)}`, options);
    return sortTenants(tenants);
  },

  async getTenantById(id: string, options?: ServiceOptions): Promise<TenantBasic> {
    return api.getOne<TenantBasic>(`${BASIC_PATH}/${id}`, options);
  },

  async createTenant(attributes: TenantBasicAttributes, options?: ServiceOptions): Promise<TenantBasic> {
    const input: CreateTenantInput = { data: { type: "tenants", attributes } };
    const response = await api.post<ApiSingleResponse<TenantBasic>>(BASIC_PATH, input, options);
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
    return { ...tenant, attributes: { ...tenant.attributes, ...updatedAttributes } };
  },

  async deleteTenant(tenantId: string, options?: ServiceOptions): Promise<void> {
    return api.delete(`${BASIC_PATH}/${tenantId}`, options);
  },

  // Tenant configuration methods (separate endpoint under /api/configurations/tenants).

  async getAllTenantConfigurations(options?: QueryOptions): Promise<TenantConfig[]> {
    const configs = await api.getList<TenantConfig>(`${CONFIG_PATH}${buildQueryString(options)}`, options);
    return sortTenants(configs).map(sortTenantConfig);
  },

  async getTenantConfigurationById(id: string, options?: ServiceOptions): Promise<TenantConfig> {
    const config = await api.getOne<TenantConfig>(`${CONFIG_PATH}/${id}`, options);
    return sortTenantConfig(config);
  },

  async createTenantConfiguration(
    tenantId: string,
    attributes: TenantConfigAttributes,
    options?: ServiceOptions,
  ): Promise<TenantConfig> {
    const input: CreateTenantConfigInput = {
      data: { id: tenantId, type: "tenants", attributes },
    };
    const response = await api.post<ApiSingleResponse<TenantConfig>>(CONFIG_PATH, input, options);
    return response.data;
  },

  async updateTenantConfiguration(
    tenant: TenantConfig,
    updatedAttributes: Partial<TenantConfigAttributes>,
    options?: ServiceOptions,
  ): Promise<TenantConfig> {
    const input: UpdateTenantConfigInput = {
      data: {
        id: tenant.id,
        type: "tenants",
        attributes: { ...tenant.attributes, ...updatedAttributes },
      },
    };
    await api.patch<void>(`${CONFIG_PATH}/${tenant.id}`, input, options);
    return { ...tenant, attributes: { ...tenant.attributes, ...updatedAttributes } };
  },

  createTenantFromTemplate(template: { attributes: TenantConfigAttributes }): TenantConfigAttributes {
    return JSON.parse(JSON.stringify(template.attributes));
  },
};

export type { TenantBasic, TenantBasicAttributes, TenantConfig, TenantConfigAttributes };
