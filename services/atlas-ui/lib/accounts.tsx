import {Tenant} from "@/app/tenants/columns";

export interface Account {
    id: string;
    attributes: {
        name: string;
        pin: string;
        pic: string;
        loggedIn: number;
        lastLogin: number;
        gender: number;
        banned: boolean;
        tos: boolean;
        language: string;
        country: string;
        characterSlots: number;
    };
}

export async function fetchAccounts(tenant: Tenant): Promise<Account[]> {
    const headers = new Headers();
    headers.set("TENANT_ID", tenant?.id);
    headers.set("REGION", tenant?.attributes.region);
    headers.set("MAJOR_VERSION", String(tenant?.attributes.majorVersion));
    headers.set("MINOR_VERSION", String(tenant?.attributes.minorVersion));

    const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || "http://localhost:3000";
    const response = await fetch(rootUrl + "/api/accounts/", {
        method: "GET",
        headers: headers,
    });
    if (!response.ok) {
        throw new Error("Failed to fetch templates.");
    }
    const responseData = await response.json();
    return responseData.data;
}