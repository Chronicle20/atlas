# Type System Patterns

## Overview

TypeScript strict mode is enabled with enhanced checks. All types live in `types/` — domain models in `types/models/` and API types in `types/api/`.

## JSON:API Model Structure

**All domain models follow the JSON:API format:**

```typescript
// types/models/character.ts
export interface Character {
  id: string;
  attributes: CharacterAttributes;
}

export interface CharacterAttributes {
  accountId: number;
  worldId: number;
  name: string;
  level: number;
  experience: number;
  hp: number;
  maxHp: number;
  mp: number;
  maxMp: number;
  meso: number;
  jobId: number;
  mapId: number;
  // ...
}
```

**Pattern:**
- `id` is always `string`
- `attributes` contains all data fields
- Nested types for complex attributes
- Optional `relationships` for related resources (e.g., Shop → Commodities)

## Enum + Label Map Pattern

For numeric enums that need display labels:

```typescript
// types/models/ban.ts
export enum BanType {
  IP = 0,
  HWID = 1,
  Account = 2,
}

export const BanTypeLabels: Record<BanType, string> = {
  [BanType.IP]: 'IP Address',
  [BanType.HWID]: 'Hardware ID',
  [BanType.Account]: 'Account',
};

export enum BanReasonCode {
  Unspecified = 0,
  Spamming = 1,
  Hacking = 2,
  TermsViolation = 3,
  Harassment = 4,
  Other = 5,
}

export const BanReasonCodeLabels: Record<BanReasonCode, string> = {
  [BanReasonCode.Unspecified]: 'Unspecified',
  [BanReasonCode.Spamming]: 'Spamming',
  // ...
};
```

## Helper Functions on Models

Attach domain logic as standalone functions alongside the model:

```typescript
// types/models/ban.ts
export function isBanExpired(ban: Ban): boolean {
  if (ban.attributes.permanent) return false;
  return new Date(ban.attributes.expiresAt) <= new Date();
}

export function isBanActive(ban: Ban): boolean {
  return !isBanExpired(ban);
}

export function formatBanExpiration(ban: Ban): string {
  if (ban.attributes.permanent) return "Permanent";
  return new Date(ban.attributes.expiresAt).toLocaleString();
}
```

## API Response Types

```typescript
// types/api/responses.ts
export interface ApiResponse<T = unknown> {
  data: T;
}

export interface ApiListResponse<T = unknown> extends ApiResponse<T[]> {
  data: T[];
}

export interface ApiSingleResponse<T = unknown> extends ApiResponse<T> {
  data: T;
}

export interface ApiErrorResponse {
  error: {
    detail: string;
    status?: number;
    code?: string;
  };
}

// Type guards
export function isApiErrorResponse(response: unknown): response is ApiErrorResponse {
  return typeof response === 'object' && response !== null && 'error' in response;
}
```

## Error Type Hierarchy

```typescript
// types/api/errors.ts
export interface ApiError {
  message: string;
  statusCode: number;
  code: string;
  details?: Record<string, unknown>;
}

export interface NetworkError extends ApiError { code: 'NETWORK_ERROR'; }
export interface ValidationError extends ApiError { code: 'VALIDATION_ERROR'; statusCode: 400 | 422; }
export interface AuthenticationError extends ApiError { code: 'AUTHENTICATION_ERROR'; statusCode: 401; }
export interface NotFoundError extends ApiError { code: 'NOT_FOUND'; statusCode: 404; }
export interface ServerError extends ApiError { code: 'SERVER_ERROR'; statusCode: 500 | 502 | 503 | 504; }

export type ApiErrorType =
  | NetworkError
  | ValidationError
  | AuthenticationError
  | NotFoundError
  | ServerError;

// Type guards
export function isNetworkError(error: unknown): error is NetworkError { /* ... */ }
export function isNotFoundError(error: unknown): error is NotFoundError { /* ... */ }
// etc.
```

## Result Pattern

```typescript
export type Result<T, E = ApiErrorType> =
  | { success: true; data: T }
  | { success: false; error: E };

export function createSuccessResult<T>(data: T): Result<T> {
  return { success: true, data };
}

export function createErrorResult<E>(error: E): Result<never, E> {
  return { success: false, error };
}
```

## Type Guard Pattern (Service Layer)

Services use private type guards for runtime checking:

```typescript
private isBan(data: unknown): data is Ban {
  return (
    typeof data === 'object' &&
    data !== null &&
    'id' in data &&
    'attributes' in data &&
    typeof (data as Ban).attributes === 'object' &&
    'banType' in (data as Ban).attributes
  );
}
```

## Update Data Types

Separate types for update payloads (partial attributes):

```typescript
export interface UpdateCharacterData {
  mapId?: number;
}

export interface CreateBanRequest {
  banType: BanType;
  value: string;
  reason: string;
  reasonCode: BanReasonCode;
  permanent: boolean;
  expiresAt: string;
  issuedBy: string;
}
```

## Type Re-exports

Types are re-exported from `services/api/index.ts` for convenient imports:

```typescript
// In consuming code, import from services:
import { type Ban, type CreateBanRequest } from "@/services/api";

// Or from types directly:
import { type Ban } from "@/types/models/ban";
```

## TypeScript Strict Mode Features

The project uses enhanced TypeScript checks:

```json
{
  "strict": true,
  "noUncheckedIndexedAccess": true,       // arr[0] is T | undefined
  "exactOptionalPropertyTypes": true,     // { x?: string } means string, not string | undefined
  "noImplicitOverride": true              // Must use override keyword
}
```

**Implications:**
- Always check array index access results for `undefined`
- Use `!` assertion only when you've already validated
- Use `override` keyword when overriding base class methods
