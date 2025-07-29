import type { ApiSingleResponse } from "@/types/api/responses";
import { api } from "@/lib/api/client";

// Lightweight tenant attributes for the new API
export type TenantBasicAttributes = {
    name: string;
    region: string;
    majorVersion: number;
    minorVersion: number;
};

// Lightweight tenant type for the new API
export type TenantBasic = {
    id: string;
    attributes: TenantBasicAttributes;
};

// Full tenant configuration attributes for the existing API
export type TenantConfigAttributes = {
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
    }
    worlds: {
        name: string;
        flag: string;
        serverMessage: string;
        eventMessage: string;
        whyAmIRecommended: string;
    }[];
}

// Full tenant configuration type for the existing API
export type TenantConfig = {
    id: string;
    attributes: TenantConfigAttributes;
};

// For backward compatibility
export type TenantAttributes = TenantConfigAttributes;
export type Tenant = TenantBasic;

// Fetch lightweight tenant list using the new API
export async function fetchTenants(): Promise<Tenant[]> {
    const tenants = await api.getList<Tenant>("/api/tenants");
    return tenants.sort((a: Tenant, b: Tenant) => {
        if (a.attributes.region !== b.attributes.region) {
            return a.attributes.region.localeCompare(b.attributes.region);
        }
        if (a.attributes.majorVersion !== b.attributes.majorVersion) {
            return a.attributes.majorVersion - b.attributes.majorVersion;
        }
        return a.attributes.minorVersion - b.attributes.minorVersion;
    });
}

// Fetch a single tenant by ID using the new API
export async function fetchTenant(tenantId: string): Promise<Tenant> {
    return api.getOne<Tenant>(`/api/tenants/${tenantId}`);
}

// Fetch tenant configuration using the existing API
export async function fetchTenantConfigurations(): Promise<TenantConfig[]> {
    const tenants = await api.getList<TenantConfig>("/api/configurations/tenants");
    return tenants
        .map((tenant: TenantConfig) => ({
            ...tenant,
            attributes: {
                ...tenant.attributes,
                socket: {
                    ...tenant.attributes.socket,
                    handlers: [...tenant.attributes.socket.handlers].sort(
                        (a, b) => parseInt(a.opCode, 16) - parseInt(b.opCode, 16)
                    ),
                    writers: [...tenant.attributes.socket.writers].sort(
                        (a, b) => parseInt(a.opCode, 16) - parseInt(b.opCode, 16)
                    ),
                },
            },
        }))
        .sort((a: TenantConfig, b: TenantConfig) => {
            if (a.attributes.region !== b.attributes.region) {
                return a.attributes.region.localeCompare(b.attributes.region);
            }
            if (a.attributes.majorVersion !== b.attributes.majorVersion) {
                return a.attributes.majorVersion - b.attributes.majorVersion;
            }
            return a.attributes.minorVersion - b.attributes.minorVersion;
        });
}

// Fetch a single tenant configuration by ID using the existing API
export async function fetchTenantConfiguration(tenantId: string): Promise<TenantConfig> {
    const tenant = await api.getOne<TenantConfig>(`/api/configurations/tenants/${tenantId}`);

    // Sort handlers and writers
    if (tenant.attributes.socket) {
        tenant.attributes.socket.handlers = [...tenant.attributes.socket.handlers].sort(
            (a, b) => parseInt(a.opCode, 16) - parseInt(b.opCode, 16)
        );
        tenant.attributes.socket.writers = [...tenant.attributes.socket.writers].sort(
            (a, b) => parseInt(a.opCode, 16) - parseInt(b.opCode, 16)
        );
    }

    return tenant;
}

// Update a tenant using the new API
export const updateTenant = async (tenant: Tenant | undefined, updatedAttributes: Partial<TenantBasicAttributes>) => {
    if (!tenant) return;

    const input = {
        data: {
            id: tenant.id,
            type: "tenants",
            attributes: {
                ...tenant.attributes,
                ...updatedAttributes,
            },
        },
    };

    await api.patch(`/api/tenants/${tenant.id}`, input);

    // If the request is successful, update the local tenant state
    return {...tenant, attributes: {...tenant.attributes, ...updatedAttributes}};
};

// Delete a tenant using the new API
export const deleteTenant = async (tenantId: string) => {
    await api.delete(`/api/tenants/${tenantId}`);
    return true;
};

// Create a tenant using the new API
export const createTenant = async (attributes: TenantBasicAttributes) => {
    const input = {
        data: {
            type: "tenants",
            attributes: attributes,
        },
    };

    const response = await api.post<ApiSingleResponse<Tenant>>("/api/tenants", input);
    return response.data;
};

// Update a tenant configuration using the existing API
export const updateTenantConfiguration = async (tenant: TenantConfig | undefined, updatedAttributes: Partial<TenantConfigAttributes>) => {
    if (!tenant) return;

    const input = {
        data: {
            id: tenant.id,
            type: "tenants",
            attributes: {
                ...tenant.attributes,
                ...updatedAttributes,
            },
        },
    };

    await api.patch(`/api/configurations/tenants/${tenant.id}`, input);

    // If the request is successful, update the local tenant state
    return {...tenant, attributes: {...tenant.attributes, ...updatedAttributes}};
};

// Create a tenant configuration using the existing API
export const createTenantConfiguration = async (attributes: TenantConfigAttributes) => {
    const input = {
        data: {
            type: "tenants",
            attributes: attributes,
        },
    };

    const response = await api.post<ApiSingleResponse<TenantConfig>>("/api/configurations/tenants", input);
    return response.data;
};

export const createTenantFromTemplate = (template: import("@/lib/templates").Template): TenantAttributes => {
    // Create a deep copy of the template attributes
    const tenantAttributes: TenantAttributes = JSON.parse(JSON.stringify(template.attributes));

    return tenantAttributes;
};
