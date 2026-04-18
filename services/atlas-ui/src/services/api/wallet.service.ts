import { api } from "@/lib/api/client";
import type { ServiceOptions } from "@/lib/api/query-params";
import type { Wallet, WalletAttributes } from "@/types/models/wallet";

const BASE_PATH = "/api/accounts";

export const walletService = {
  async getWallet(accountId: string, options?: ServiceOptions): Promise<Wallet> {
    return api.getOne<Wallet>(`${BASE_PATH}/${accountId}/wallet`, options);
  },

  async updateWallet(
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
