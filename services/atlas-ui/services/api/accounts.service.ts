/**
 * Accounts Service
 * 
 * Provides comprehensive account management functionality including:
 * - Account retrieval operations with tenant context
 * - Account session management
 * - Enhanced error handling and caching
 * - Request cancellation and deduplication support
 */

import { BaseService, type ServiceOptions, type QueryOptions, type ValidationError } from './base.service';
import type { Account, AccountAttributes } from '@/types/models/account';
import type { Tenant } from '@/types/models/tenant';
import { api } from '@/lib/api/client';

/**
 * Account-specific query options
 */
interface AccountQueryOptions extends QueryOptions {
  /** Filter by account name */
  name?: string;
  /** Filter by banned status */
  banned?: boolean;
  /** Filter by logged in status */
  loggedIn?: boolean;
  /** Filter by language */
  language?: string;
  /** Filter by country */
  country?: string;
}

/**
 * Accounts service class extending BaseService with account-specific functionality
 */
class AccountsService extends BaseService {
  protected basePath = '/api/accounts';

  /**
   * Validate account data before API calls
   */
  protected override validate<T>(data: T): ValidationError[] {
    const errors: ValidationError[] = [];

    if (this.isAccountAttributes(data)) {
      if (!data.name || data.name.trim().length === 0) {
        errors.push({ field: 'name', message: 'Account name is required' });
      }
      if (data.name && data.name.length > 12) {
        errors.push({ field: 'name', message: 'Account name must be 12 characters or less' });
      }
      if (!data.pin || data.pin.trim().length === 0) {
        errors.push({ field: 'pin', message: 'PIN is required' });
      }
      if (data.pin && data.pin.length > 8) {
        errors.push({ field: 'pin', message: 'PIN must be 8 characters or less' });
      }
      if (!data.pic || data.pic.trim().length === 0) {
        errors.push({ field: 'pic', message: 'PIC is required' });
      }
      if (data.pic && data.pic.length > 8) {
        errors.push({ field: 'pic', message: 'PIC must be 8 characters or less' });
      }
      if (data.characterSlots < 0 || data.characterSlots > 30) {
        errors.push({ field: 'characterSlots', message: 'Character slots must be between 0 and 30' });
      }
      if (data.gender < 0 || data.gender > 1) {
        errors.push({ field: 'gender', message: 'Gender must be 0 (male) or 1 (female)' });
      }
    }

    return errors;
  }

  /**
   * Transform response data to ensure consistent structure
   */
  protected override transformResponse<T>(data: T): T {
    if (this.isAccount(data)) {
      // Ensure all numeric fields are properly typed
      const transformed = { ...data };
      transformed.attributes = {
        ...transformed.attributes,
        loggedIn: Number(transformed.attributes.loggedIn),
        lastLogin: Number(transformed.attributes.lastLogin),
        gender: Number(transformed.attributes.gender),
        characterSlots: Number(transformed.attributes.characterSlots),
        banned: Boolean(transformed.attributes.banned),
        tos: Boolean(transformed.attributes.tos),
      };
      return transformed as T;
    }
    return data;
  }

  /**
   * Sort accounts by name (case-insensitive)
   */
  private sortAccounts(accounts: Account[]): Account[] {
    return accounts.sort((a, b) => 
      a.attributes.name.toLowerCase().localeCompare(b.attributes.name.toLowerCase())
    );
  }

  /**
   * Get all accounts for a specific tenant with sorting and filtering
   */
  async getAllAccounts(tenant: Tenant, options?: AccountQueryOptions): Promise<Account[]> {
    // Set tenant context for this request
    api.setTenant(tenant);
    
    // Build query options with account-specific filters
    const queryOptions: QueryOptions = { ...options };
    if (options?.name) {
      queryOptions.filters = { ...queryOptions.filters, name: options.name };
    }
    if (options?.banned !== undefined) {
      queryOptions.filters = { ...queryOptions.filters, banned: options.banned };
    }
    if (options?.loggedIn !== undefined) {
      queryOptions.filters = { ...queryOptions.filters, loggedIn: options.loggedIn };
    }
    if (options?.language) {
      queryOptions.filters = { ...queryOptions.filters, language: options.language };
    }
    if (options?.country) {
      queryOptions.filters = { ...queryOptions.filters, country: options.country };
    }

    const accounts = await this.getAll<Account>(queryOptions);
    return this.sortAccounts(accounts);
  }

  /**
   * Get account by ID for a specific tenant
   */
  async getAccountById(tenant: Tenant, id: string, options?: ServiceOptions): Promise<Account> {
    // Set tenant context for this request
    api.setTenant(tenant);
    
    return this.getById<Account>(id, options);
  }

  /**
   * Check if an account exists for a specific tenant
   */
  async accountExists(tenant: Tenant, id: string, options?: ServiceOptions): Promise<boolean> {
    // Set tenant context for this request
    api.setTenant(tenant);
    
    return this.exists(id, options);
  }

  /**
   * Get accounts by name pattern (case-insensitive search)
   */
  async searchAccountsByName(tenant: Tenant, namePattern: string, options?: ServiceOptions): Promise<Account[]> {
    const queryOptions: AccountQueryOptions = {
      ...options,
      search: namePattern,
      name: namePattern
    };
    
    return this.getAllAccounts(tenant, queryOptions);
  }

  /**
   * Get logged-in accounts for a specific tenant
   */
  async getLoggedInAccounts(tenant: Tenant, options?: ServiceOptions): Promise<Account[]> {
    const queryOptions: AccountQueryOptions = {
      ...options,
      loggedIn: true
    };
    
    return this.getAllAccounts(tenant, queryOptions);
  }

  /**
   * Get banned accounts for a specific tenant
   */
  async getBannedAccounts(tenant: Tenant, options?: ServiceOptions): Promise<Account[]> {
    const queryOptions: AccountQueryOptions = {
      ...options,
      banned: true
    };
    
    return this.getAllAccounts(tenant, queryOptions);
  }

  /**
   * Terminate account session - Force logout an account
   */
  async terminateAccountSession(tenant: Tenant, accountId: string, options?: ServiceOptions): Promise<void> {
    // Set tenant context for this request
    api.setTenant(tenant);
    
    const processedOptions = options ? { ...options } : {};
    
    return api.delete(`${this.basePath}/${accountId}/session`, processedOptions);
  }

  /**
   * Get account statistics for a tenant
   */
  async getAccountStats(tenant: Tenant, options?: ServiceOptions): Promise<{
    total: number;
    loggedIn: number;
    banned: number;
    totalCharacterSlots: number;
    averageCharacterSlots: number;
  }> {
    const accounts = await this.getAllAccounts(tenant, options);
    
    const stats = {
      total: accounts.length,
      loggedIn: accounts.filter(acc => acc.attributes.loggedIn > 0).length,
      banned: accounts.filter(acc => acc.attributes.banned).length,
      totalCharacterSlots: accounts.reduce((sum, acc) => sum + acc.attributes.characterSlots, 0),
      averageCharacterSlots: 0
    };
    
    stats.averageCharacterSlots = stats.total > 0 ? stats.totalCharacterSlots / stats.total : 0;
    
    return stats;
  }

  /**
   * Batch terminate multiple account sessions
   */
  async terminateMultipleSessions(
    tenant: Tenant,
    accountIds: string[],
    options?: ServiceOptions
  ): Promise<{ successful: string[]; failed: Array<{ id: string; error: string }> }> {
    // Set tenant context for this request
    api.setTenant(tenant);
    
    const successful: string[] = [];
    const failed: Array<{ id: string; error: string }> = [];
    
    // Process terminations with limited concurrency to avoid overwhelming the server
    const concurrency = 3;
    for (let i = 0; i < accountIds.length; i += concurrency) {
      const batch = accountIds.slice(i, i + concurrency);
      
      const promises = batch.map(async (accountId): Promise<{ success: true; accountId: string } | { success: false; accountId: string; error: string }> => {
        try {
          await this.terminateAccountSession(tenant, accountId, options);
          return { success: true, accountId };
        } catch (error) {
          return { 
            success: false,
            accountId,
            error: error instanceof Error ? error.message : 'Unknown error'
          };
        }
      });
      
      const results = await Promise.all(promises);
      
      results.forEach(result => {
        if (result.success) {
          successful.push(result.accountId);
        } else {
          failed.push({ id: result.accountId, error: result.error });
        }
      });
    }
    
    return { successful, failed };
  }

  // === TYPE GUARDS ===

  private isAccount(data: unknown): data is Account {
    return (
      typeof data === 'object' &&
      data !== null &&
      'id' in data &&
      'attributes' in data &&
      typeof (data as any).attributes === 'object' &&
      'name' in (data as any).attributes
    );
  }

  private isAccountAttributes(data: unknown): data is AccountAttributes {
    return (
      typeof data === 'object' &&
      data !== null &&
      'name' in data &&
      'pin' in data &&
      'pic' in data &&
      'characterSlots' in data
    );
  }
}

// Create and export a singleton instance
export const accountsService = new AccountsService();

// Export types for use in other files
export type { Account, AccountAttributes, AccountQueryOptions };