import type {Tenant} from "@/types/models/tenant";
import {tenantHeaders} from "@/lib/headers";

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
    const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
    const response = await fetch(rootUrl + "/api/accounts/", {
        method: "GET",
        headers: tenantHeaders(tenant),
    });
    if (!response.ok) {
        throw new Error("Failed to fetch accounts.");
    }
    const responseData = await response.json();
    return responseData.data;
}

export async function terminateAccountSession(tenant: Tenant, accountId: string): Promise<void> {
    const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
    const response = await fetch(rootUrl + "/api/accounts/" + accountId + "/session", {
        method: "DELETE",
        headers: tenantHeaders(tenant),
    });
    if (!response.ok) {
        throw new Error("Failed to delete account session.");
    }
    return Promise.resolve();
}