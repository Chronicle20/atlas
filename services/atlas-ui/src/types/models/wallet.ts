export interface Wallet {
    id: string;
    attributes: WalletAttributes;
}

export interface WalletAttributes {
    accountId: number;
    credit: number;
    points: number;
    prepaid: number;
}
