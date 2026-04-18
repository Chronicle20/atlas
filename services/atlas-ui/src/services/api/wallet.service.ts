import { BaseService, type ServiceOptions } from './base.service';
import type { Wallet, WalletAttributes } from '@/types/models/wallet';
import type { Tenant } from '@/types/models/tenant';
import { api } from '@/lib/api/client';

class WalletService extends BaseService {
  protected basePath = '/api/accounts';

  async getWallet(tenant: Tenant, accountId: string, options?: ServiceOptions): Promise<Wallet> {
    const processedOptions = options ? { ...options } : {};
    return api.getOne<Wallet>(`${this.basePath}/${accountId}/wallet`, processedOptions);
  }

  async updateWallet(
    tenant: Tenant,
    accountId: string,
    credit: number,
    points: number,
    prepaid: number,
    options?: ServiceOptions
  ): Promise<Wallet> {
    const body = {
      data: {
        type: 'wallets',
        attributes: { credit, points, prepaid },
      },
    };
    const processedOptions = options ? { ...options } : {};
    return api.patch<Wallet>(`${this.basePath}/${accountId}/wallet`, body, processedOptions);
  }
}

export const walletService = new WalletService();

export type { Wallet, WalletAttributes };
