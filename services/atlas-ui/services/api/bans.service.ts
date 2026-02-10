/**
 * Bans Service
 *
 * Provides comprehensive ban management functionality including:
 * - Ban retrieval operations with tenant context
 * - Ban creation and deletion
 * - Ban checking for IP, HWID, and account
 * - Enhanced error handling and caching
 */

import { BaseService, type ServiceOptions, type QueryOptions, type ValidationError } from './base.service';
import type { Ban, BanAttributes, CreateBanRequest, CheckBanResult, BanType } from '@/types/models/ban';
import type { Tenant } from '@/types/models/tenant';
import { api } from '@/lib/api/client';

/**
 * Ban-specific query options
 */
interface BanQueryOptions extends QueryOptions {
    /** Filter by ban type */
    type?: BanType;
}

/**
 * Parameters for checking if a value is banned
 */
interface CheckBanParams {
    ip?: string;
    hwid?: string;
    accountId?: number;
}

/**
 * Bans service class extending BaseService with ban-specific functionality
 */
class BansService extends BaseService {
    protected basePath = '/api/bans';

    /**
     * Validate ban data before API calls
     */
    protected override validate<T>(data: T): ValidationError[] {
        const errors: ValidationError[] = [];

        if (this.isCreateBanRequest(data)) {
            if (!data.value || data.value.trim().length === 0) {
                errors.push({ field: 'value', message: 'Ban value is required' });
            }
            if (data.banType === 0 && data.value) { // IP type
                // Basic IP/CIDR validation
                const ipRegex = /^(\d{1,3}\.){3}\d{1,3}(\/\d{1,2})?$/;
                if (!ipRegex.test(data.value)) {
                    errors.push({ field: 'value', message: 'Invalid IP address or CIDR format' });
                }
            }
            if (!data.permanent && (!data.expiresAt || new Date(data.expiresAt) <= new Date())) {
                errors.push({ field: 'expiresAt', message: 'Expiration date must be in the future for non-permanent bans' });
            }
        }

        return errors;
    }

    /**
     * Transform response data to ensure consistent structure
     */
    protected override transformResponse<T>(data: T): T {
        if (this.isBan(data)) {
            const transformed = { ...data };
            transformed.attributes = {
                ...transformed.attributes,
                banType: Number(transformed.attributes.banType),
                reasonCode: Number(transformed.attributes.reasonCode),
                permanent: Boolean(transformed.attributes.permanent),
                expiresAt: String(transformed.attributes.expiresAt),
            };
            return transformed as T;
        }
        return data;
    }

    /**
     * Sort bans by ID (newest first)
     */
    private sortBans(bans: Ban[]): Ban[] {
        return bans.sort((a, b) => Number(b.id) - Number(a.id));
    }

    /**
     * Get all bans for a specific tenant with optional type filter
     */
    async getAllBans(tenant: Tenant, options?: BanQueryOptions): Promise<Ban[]> {
        api.setTenant(tenant);

        let url = this.basePath;
        if (options?.type !== undefined) {
            url += `?type=${options.type}`;
        }

        const bans = await api.getList<Ban>(url, options);
        return this.sortBans(bans.map(item => this.transformResponse(item)));
    }

    /**
     * Get ban by ID for a specific tenant
     */
    async getBanById(tenant: Tenant, id: string, options?: ServiceOptions): Promise<Ban> {
        api.setTenant(tenant);
        return this.getById<Ban>(id, options);
    }

    /**
     * Check if a ban exists for a specific tenant
     */
    async banExists(tenant: Tenant, id: string, options?: ServiceOptions): Promise<boolean> {
        api.setTenant(tenant);
        return this.exists(id, options);
    }

    /**
     * Create a new ban
     */
    async createBan(tenant: Tenant, data: CreateBanRequest, options?: ServiceOptions): Promise<Ban> {
        api.setTenant(tenant);

        // Validate the data
        const validationErrors = this.validate(data);
        if (validationErrors.length > 0) {
            throw new Error(`Validation failed: ${validationErrors.map(e => e.message).join(', ')}`);
        }

        const response = await api.post<{ data: Ban }>(this.basePath, data, options);
        return this.transformResponse(response.data);
    }

    /**
     * Delete a ban by ID
     */
    async deleteBan(tenant: Tenant, id: string, options?: ServiceOptions): Promise<void> {
        api.setTenant(tenant);
        return this.delete(id, options);
    }

    /**
     * Check if a value is banned (IP, HWID, or account ID)
     */
    async checkBan(tenant: Tenant, params: CheckBanParams, options?: ServiceOptions): Promise<CheckBanResult> {
        api.setTenant(tenant);

        const queryParams = new URLSearchParams();
        if (params.ip) queryParams.append('ip', params.ip);
        if (params.hwid) queryParams.append('hwid', params.hwid);
        if (params.accountId) queryParams.append('accountId', params.accountId.toString());

        const url = `${this.basePath}/check?${queryParams.toString()}`;
        const response = await api.get<{ data: CheckBanResult }>(url, options);
        return response.data;
    }

    /**
     * Get bans by type
     */
    async getBansByType(tenant: Tenant, type: BanType, options?: ServiceOptions): Promise<Ban[]> {
        return this.getAllBans(tenant, { ...options, type });
    }

    // === TYPE GUARDS ===

    private isBan(data: unknown): data is Ban {
        return (
            typeof data === 'object' &&
            data !== null &&
            'id' in data &&
            'attributes' in data &&
            typeof (data as Ban).attributes === 'object' &&
            'banType' in (data as Ban).attributes &&
            'value' in (data as Ban).attributes
        );
    }

    private isCreateBanRequest(data: unknown): data is CreateBanRequest {
        return (
            typeof data === 'object' &&
            data !== null &&
            'banType' in data &&
            'value' in data
        );
    }
}

// Create and export a singleton instance
export const bansService = new BansService();

// Export types for use in other files
export type { Ban, BanAttributes, CreateBanRequest, CheckBanResult, BanQueryOptions, CheckBanParams };
