import type {Tenant} from "@/types/models/tenant";
import { api } from "@/lib/api/client";

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
    api.setTenant(tenant);
    return api.getList<Account>('/api/accounts/');
}

export async function terminateAccountSession(tenant: Tenant, accountId: string): Promise<void> {
    api.setTenant(tenant);
    return api.delete<void>(`/api/accounts/${accountId}/session`);
}