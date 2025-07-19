import type {ApiListResponse, ApiSingleResponse} from "@/types/api/responses";
import {createApiErrorFromResponse, createErrorFromUnknown} from "@/types/api/errors";

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
    try {
        const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
        const response = await fetch(rootUrl + "/api/tenants");
        if (!response.ok) {
            throw createApiErrorFromResponse(response.status, "Failed to fetch tenants");
        }

        const responseData: ApiListResponse<Tenant> = await response.json();
        return responseData.data
            .sort((a: Tenant, b: Tenant) => {
                if (a.attributes.region !== b.attributes.region) {
                    return a.attributes.region.localeCompare(b.attributes.region);
                }
                if (a.attributes.majorVersion !== b.attributes.majorVersion) {
                    return a.attributes.majorVersion - b.attributes.majorVersion;
                }
                return a.attributes.minorVersion - b.attributes.minorVersion;
            });
    } catch (error: unknown) {
        if (typeof error === 'object' && error !== null && 'statusCode' in error) {
            throw error; // Re-throw API errors
        }
        throw createErrorFromUnknown(error, "Failed to fetch tenants");
    }
}

// Fetch a single tenant by ID using the new API
export async function fetchTenant(tenantId: string): Promise<Tenant> {
    try {
        const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
        const response = await fetch(`${rootUrl}/api/tenants/${tenantId}`);
        if (!response.ok) {
            throw createApiErrorFromResponse(response.status, `Failed to fetch tenant with ID: ${tenantId}`);
        }

        const responseData: ApiSingleResponse<Tenant> = await response.json();
        return responseData.data;
    } catch (error: unknown) {
        if (typeof error === 'object' && error !== null && 'statusCode' in error) {
            throw error; // Re-throw API errors
        }
        throw createErrorFromUnknown(error, `Failed to fetch tenant with ID: ${tenantId}`);
    }
}

// Fetch tenant configuration using the existing API
export async function fetchTenantConfigurations(): Promise<TenantConfig[]> {
    try {
        const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
        const response = await fetch(rootUrl + "/api/configurations/tenants");
        if (!response.ok) {
            throw createApiErrorFromResponse(response.status, "Failed to fetch tenant configurations");
        }

        const responseData: ApiListResponse<TenantConfig> = await response.json();
        return responseData.data
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
    } catch (error: unknown) {
        if (typeof error === 'object' && error !== null && 'statusCode' in error) {
            throw error; // Re-throw API errors
        }
        throw createErrorFromUnknown(error, "Failed to fetch tenant configurations");
    }
}

// Fetch a single tenant configuration by ID using the existing API
export async function fetchTenantConfiguration(tenantId: string): Promise<TenantConfig> {
    try {
        const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
        const response = await fetch(`${rootUrl}/api/configurations/tenants/${tenantId}`);
        if (!response.ok) {
            throw createApiErrorFromResponse(response.status, `Failed to fetch tenant configuration for ID: ${tenantId}`);
        }

        const responseData: ApiSingleResponse<TenantConfig> = await response.json();
        const tenant = responseData.data;

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
    } catch (error: unknown) {
        if (typeof error === 'object' && error !== null && 'statusCode' in error) {
            throw error; // Re-throw API errors
        }
        throw createErrorFromUnknown(error, `Failed to fetch tenant configuration for ID: ${tenantId}`);
    }
}

// Update a tenant using the new API
export const updateTenant = async (tenant: Tenant | undefined, updatedAttributes: Partial<TenantBasicAttributes>) => {
    if (!tenant) return;

    try {
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

        const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
        const response = await fetch(`${rootUrl}/api/tenants/${tenant.id}`, {
            method: "PATCH",
            headers: {"Content-Type": "application/json"},
            body: JSON.stringify(input),
        });

        if (!response.ok) {
            throw createApiErrorFromResponse(response.status, `Failed to update tenant with ID: ${tenant.id}`);
        }

        // If the request is successful, update the local tenant state
        return {...tenant, attributes: {...tenant.attributes, ...updatedAttributes}};
    } catch (error: unknown) {
        if (typeof error === 'object' && error !== null && 'statusCode' in error) {
            throw error; // Re-throw API errors
        }
        throw createErrorFromUnknown(error, `Failed to update tenant with ID: ${tenant.id}`);
    }
};

// Delete a tenant using the new API
export const deleteTenant = async (tenantId: string) => {
    try {
        const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
        const response = await fetch(`${rootUrl}/api/tenants/${tenantId}`, {
            method: "DELETE",
            headers: {"Content-Type": "application/json"},
        });

        if (!response.ok) {
            throw createApiErrorFromResponse(response.status, `Failed to delete tenant with ID: ${tenantId}`);
        }

        return true;
    } catch (error: unknown) {
        if (typeof error === 'object' && error !== null && 'statusCode' in error) {
            throw error; // Re-throw API errors
        }
        throw createErrorFromUnknown(error, `Failed to delete tenant with ID: ${tenantId}`);
    }
};

// Create a tenant using the new API
export const createTenant = async (attributes: TenantBasicAttributes) => {
    try {
        const input = {
            data: {
                type: "tenants",
                attributes: attributes,
            },
        };

        const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
        const response = await fetch(`${rootUrl}/api/tenants`, {
            method: "POST",
            headers: {"Content-Type": "application/json"},
            body: JSON.stringify(input),
        });

        if (!response.ok) {
            throw createApiErrorFromResponse(response.status, "Failed to create tenant");
        }

        const responseData: ApiSingleResponse<Tenant> = await response.json();
        return responseData.data;
    } catch (error: unknown) {
        if (typeof error === 'object' && error !== null && 'statusCode' in error) {
            throw error; // Re-throw API errors
        }
        throw createErrorFromUnknown(error, "Failed to create tenant");
    }
};

// Update a tenant configuration using the existing API
export const updateTenantConfiguration = async (tenant: TenantConfig | undefined, updatedAttributes: Partial<TenantConfigAttributes>) => {
    if (!tenant) return;

    try {
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

        const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
        const response = await fetch(`${rootUrl}/api/configurations/tenants/${tenant.id}`, {
            method: "PATCH",
            headers: {"Content-Type": "application/json"},
            body: JSON.stringify(input),
        });

        if (!response.ok) {
            throw createApiErrorFromResponse(response.status, `Failed to update tenant configuration with ID: ${tenant.id}`);
        }

        // If the request is successful, update the local tenant state
        return {...tenant, attributes: {...tenant.attributes, ...updatedAttributes}};
    } catch (error: unknown) {
        if (typeof error === 'object' && error !== null && 'statusCode' in error) {
            throw error; // Re-throw API errors
        }
        throw createErrorFromUnknown(error, `Failed to update tenant configuration with ID: ${tenant.id}`);
    }
};

// Create a tenant configuration using the existing API
export const createTenantConfiguration = async (attributes: TenantConfigAttributes) => {
    try {
        const input = {
            data: {
                type: "tenants",
                attributes: attributes,
            },
        };

        const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
        const response = await fetch(`${rootUrl}/api/configurations/tenants`, {
            method: "POST",
            headers: {"Content-Type": "application/json"},
            body: JSON.stringify(input),
        });

        if (!response.ok) {
            throw createApiErrorFromResponse(response.status, "Failed to create tenant configuration");
        }

        const responseData: ApiSingleResponse<TenantConfig> = await response.json();
        return responseData.data;
    } catch (error: unknown) {
        if (typeof error === 'object' && error !== null && 'statusCode' in error) {
            throw error; // Re-throw API errors
        }
        throw createErrorFromUnknown(error, "Failed to create tenant configuration");
    }
};

export const createTenantFromTemplate = (template: import("@/lib/templates").Template): TenantAttributes => {
    // Create a deep copy of the template attributes
    const tenantAttributes: TenantAttributes = JSON.parse(JSON.stringify(template.attributes));

    return tenantAttributes;
};
