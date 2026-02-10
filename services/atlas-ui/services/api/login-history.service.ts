/**
 * Login History Service
 *
 * Provides login history retrieval functionality including:
 * - History lookup by IP address
 * - History lookup by HWID
 * - History lookup by account ID
 */

import { BaseService, type ServiceOptions } from './base.service';
import type { LoginHistoryEntry } from '@/types/models/ban';
import type { Tenant } from '@/types/models/tenant';
import { api } from '@/lib/api/client';

/**
 * Login History service class extending BaseService
 */
class LoginHistoryService extends BaseService {
    protected basePath = '/api/history';

    /**
     * Transform response data to ensure consistent structure
     */
    protected override transformResponse<T>(data: T): T {
        if (this.isLoginHistoryEntry(data)) {
            const transformed = { ...data };
            transformed.attributes = {
                ...transformed.attributes,
                accountId: Number(transformed.attributes.accountId),
                success: Boolean(transformed.attributes.success),
            };
            return transformed as T;
        }
        return data;
    }

    /**
     * Get login history by IP address
     */
    async getByIp(tenant: Tenant, ip: string, options?: ServiceOptions): Promise<LoginHistoryEntry[]> {
        api.setTenant(tenant);

        const url = `${this.basePath}?ip=${encodeURIComponent(ip)}`;
        const entries = await api.getList<LoginHistoryEntry>(url, options);
        return entries.map(item => this.transformResponse(item));
    }

    /**
     * Get login history by HWID
     */
    async getByHwid(tenant: Tenant, hwid: string, options?: ServiceOptions): Promise<LoginHistoryEntry[]> {
        api.setTenant(tenant);

        const url = `${this.basePath}?hwid=${encodeURIComponent(hwid)}`;
        const entries = await api.getList<LoginHistoryEntry>(url, options);
        return entries.map(item => this.transformResponse(item));
    }

    /**
     * Get login history by account ID
     */
    async getByAccountId(tenant: Tenant, accountId: number, options?: ServiceOptions): Promise<LoginHistoryEntry[]> {
        api.setTenant(tenant);

        const url = `${this.basePath}/accounts/${accountId}`;
        const entries = await api.getList<LoginHistoryEntry>(url, options);
        return entries.map(item => this.transformResponse(item));
    }

    /**
     * Search login history with multiple criteria
     */
    async search(
        tenant: Tenant,
        criteria: { ip?: string; hwid?: string; accountId?: number },
        options?: ServiceOptions
    ): Promise<LoginHistoryEntry[]> {
        // Prioritize by specificity: accountId > hwid > ip
        if (criteria.accountId) {
            return this.getByAccountId(tenant, criteria.accountId, options);
        }
        if (criteria.hwid) {
            return this.getByHwid(tenant, criteria.hwid, options);
        }
        if (criteria.ip) {
            return this.getByIp(tenant, criteria.ip, options);
        }
        return [];
    }

    // === TYPE GUARDS ===

    private isLoginHistoryEntry(data: unknown): data is LoginHistoryEntry {
        return (
            typeof data === 'object' &&
            data !== null &&
            'id' in data &&
            'attributes' in data &&
            typeof (data as LoginHistoryEntry).attributes === 'object' &&
            'accountId' in (data as LoginHistoryEntry).attributes &&
            'accountName' in (data as LoginHistoryEntry).attributes
        );
    }
}

// Create and export a singleton instance
export const loginHistoryService = new LoginHistoryService();

// Export types for use in other files
export type { LoginHistoryEntry };
