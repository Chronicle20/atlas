export type TenantAttributes = {
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

export type Tenant = {
    id: string;
    attributes: TenantAttributes;
};

export async function fetchTenants(): Promise<Tenant[]> {
    const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
    const response = await fetch(rootUrl + "/api/configurations/tenants");
    if (!response.ok) {
        throw new Error("Failed to fetch templates.");
    }

    const responseData = await response.json();
    return responseData.data;
}

export const updateTenant = async (tenant: Tenant | undefined, updatedAttributes: Partial<TenantAttributes>) => {
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

    const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
    const response = await fetch(`${rootUrl}/api/configurations/tenants/${tenant.id}`, {
        method: "PATCH",
        headers: {"Content-Type": "application/json"},
        body: JSON.stringify(input),
    });

    if (!response.ok) throw new Error("Failed to submit data.");

    // If the request is successful, update the local tenant state
    return {...tenant, attributes: {...tenant.attributes, ...updatedAttributes}};
}