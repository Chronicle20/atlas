// Ban domain model types

/**
 * Ban type enumeration matching backend constants
 */
export enum BanType {
    IP = 0,
    HWID = 1,
    Account = 2
}

/**
 * Human-readable labels for ban types
 */
export const BanTypeLabels: Record<BanType, string> = {
    [BanType.IP]: 'IP Address',
    [BanType.HWID]: 'Hardware ID',
    [BanType.Account]: 'Account'
};

/**
 * Reason codes for bans
 */
export enum BanReasonCode {
    Unspecified = 0,
    Spamming = 1,
    Hacking = 2,
    TermsViolation = 3,
    Harassment = 4,
    Other = 5
}

/**
 * Human-readable labels for reason codes
 */
export const BanReasonCodeLabels: Record<BanReasonCode, string> = {
    [BanReasonCode.Unspecified]: 'Unspecified',
    [BanReasonCode.Spamming]: 'Spamming',
    [BanReasonCode.Hacking]: 'Hacking/Cheating',
    [BanReasonCode.TermsViolation]: 'Terms Violation',
    [BanReasonCode.Harassment]: 'Harassment',
    [BanReasonCode.Other]: 'Other'
};

/**
 * Ban attributes from the API
 */
export interface BanAttributes {
    banType: BanType;
    value: string;
    reason: string;
    reasonCode: BanReasonCode;
    permanent: boolean;
    expiresAt: string; // RFC3339 timestamp from Go backend
    issuedBy: string;
}

/**
 * Ban model following JSON:API structure
 */
export interface Ban {
    id: string;
    attributes: BanAttributes;
}

/**
 * Request body for creating a new ban
 */
export interface CreateBanRequest {
    banType: BanType;
    value: string;
    reason: string;
    reasonCode: BanReasonCode;
    permanent: boolean;
    expiresAt: string;
    issuedBy: string;
}

/**
 * Result of a ban check
 */
export interface CheckBanResult {
    id: string;
    attributes: CheckBanAttributes;
}

export interface CheckBanAttributes {
    banned: boolean;
    banType?: BanType;
    reason?: string;
    reasonCode?: BanReasonCode;
    permanent?: boolean;
    expiresAt?: string;
}

/**
 * Login history entry attributes
 */
export interface LoginHistoryAttributes {
    accountId: number;
    accountName: string;
    ipAddress: string;
    hwid: string;
    success: boolean;
    failureReason?: string;
}

/**
 * Login history entry following JSON:API structure
 */
export interface LoginHistoryEntry {
    id: string;
    attributes: LoginHistoryAttributes;
}

/**
 * Check if a time string represents Go's zero value
 */
export function isZeroTime(t: string): boolean {
    return !t || t === '0001-01-01T00:00:00Z';
}

/**
 * Helper function to check if a ban is expired
 */
export function isBanExpired(ban: Ban): boolean {
    if (ban.attributes.permanent) {
        return false;
    }
    return !isZeroTime(ban.attributes.expiresAt) && new Date(ban.attributes.expiresAt) < new Date();
}

/**
 * Helper function to check if a ban is active
 */
export function isBanActive(ban: Ban): boolean {
    return !isBanExpired(ban);
}

/**
 * Helper function to format ban expiration
 */
export function formatBanExpiration(ban: Ban): string {
    if (ban.attributes.permanent) {
        return 'Never (Permanent)';
    }
    if (isZeroTime(ban.attributes.expiresAt)) {
        return 'Not set';
    }
    return new Date(ban.attributes.expiresAt).toLocaleString();
}
