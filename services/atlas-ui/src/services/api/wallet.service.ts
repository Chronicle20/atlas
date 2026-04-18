import { api } from "@/lib/api/client";
import type { ServiceOptions } from "@/lib/api/query-params";
import type { Wallet, WalletAttributes } from "@/types/models/wallet";
import type { Tenant } from "@/types/models/tenant";

const BASE_PATH = "/api/accounts";

export const walletService = {
  async getWallet(_tenant: Tenant, accountId: string, options?: ServiceOptions): Promise<Wallet> {
    return api.getOne<Wallet>(`${BASE_PATH}/${accountId}/wallet`, options);
  },

  async updateWallet(
    _tenant: Tenant,
    accountId: string,
    credit: number,
    points: number,
    prepaid: number,
    options?: ServiceOptions,
  ): Promise<Wallet> {
    const body = {
      data: {
        type: "wallets",
        attributes: { credit, points, prepaid },
      },
    };
    return api.patch<Wallet>(`${BASE_PATH}/${accountId}/wallet`, body, options);
  },
};

export type { Wallet, WalletAttributes };
