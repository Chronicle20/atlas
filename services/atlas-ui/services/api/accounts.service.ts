/**
 * Account management service
 * Handles all account-related API operations
 */
import { BaseService } from './base.service';
import type { Account, CreateAccountDto } from '@/types/models/account';

class AccountsService extends BaseService {
  protected basePath = '/accounts';

  /**
   * Get all accounts
   */
  async getAll(signal?: AbortSignal): Promise<Account[]> {
    return super.getAll<Account>(signal);
  }

  /**
   * Get account by ID
   */
  async getById(id: string, signal?: AbortSignal): Promise<Account> {
    return super.getById<Account>(id, signal);
  }

  /**
   * Create new account
   */
  async create(data: CreateAccountDto, signal?: AbortSignal): Promise<Account> {
    return super.create<Account, CreateAccountDto>(data, signal);
  }

  /**
   * Update existing account
   */
  async update(id: string, data: Partial<Account>, signal?: AbortSignal): Promise<Account> {
    return super.update<Account, Partial<Account>>(id, data, signal);
  }

  /**
   * Delete account
   */
  async delete(id: string, signal?: AbortSignal): Promise<void> {
    return super.delete(id, signal);
  }
}

export const accountsService = new AccountsService();