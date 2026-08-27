// Account domain model types
// Re-exported from lib/accounts.tsx to centralize type definitions

export interface Account {
  id: string;
  attributes: AccountAttributes;
}

export interface AccountAttributes {
  name: string;
  pin: string;
  pic: string;
  pinAttempts: number;
  picAttempts: number;
  /**
   * yyyymmdd as an integer, 0 when unset — NOT a timestamp. See
   * lib/utils/birth-date.ts for the conversion helpers.
   */
  birthDate: number;
  loggedIn: number;
  lastLogin: number;
  gender: number;
  tos: boolean;
  language: string;
  country: string;
}
