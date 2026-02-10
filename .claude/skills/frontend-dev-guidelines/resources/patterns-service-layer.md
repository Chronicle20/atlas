# Service Layer Patterns

## Overview

The service layer (`services/api/`) provides typed abstractions over the API client. Services are **singletons** — instantiated once and exported as module-level constants.

## Two Service Patterns

### 1. BaseService Pattern (Preferred for Complex Resources)

Extend `BaseService` for resources that need validation, transformation, or batch operations.

```typescript
// services/api/bans.service.ts
class BansService extends BaseService {
  protected basePath = '/api/bans';

  // Override validation for create/update
  protected override validate<T>(data: T): ValidationError[] {
    const errors: ValidationError[] = [];
    if (this.isCreateBanRequest(data)) {
      if (!data.value || data.value.trim().length === 0) {
        errors.push({ field: 'value', message: 'Ban value is required' });
      }
    }
    return errors;
  }

  // Override response transformation
  protected override transformResponse<T>(data: T): T {
    if (this.isBan(data)) {
      return {
        ...data,
        attributes: {
          ...data.attributes,
          banType: Number(data.attributes.banType),
          permanent: Boolean(data.attributes.permanent),
        },
      } as T;
    }
    return data;
  }

  // Public methods with tenant injection
  async getAllBans(tenant: Tenant, options?: QueryOptions): Promise<Ban[]> {
    api.setTenant(tenant);
    const bans = await api.getList<Ban>(this.basePath, options);
    return bans.map(item => this.transformResponse(item));
  }

  // Type guard (private)
  private isBan(data: unknown): data is Ban {
    return typeof data === 'object' && data !== null
      && 'id' in data && 'attributes' in data;
  }
}

export const bansService = new BansService();
```

### 2. Direct API Client Pattern (Simple Resources)

For services without validation or transformation needs.

```typescript
// services/api/characters.service.ts
class CharactersService {
  private basePath = '/api/characters';

  async getAll(tenant: Tenant, options?: ServiceOptions): Promise<Character[]> {
    api.setTenant(tenant);
    return api.getList<Character>(this.basePath, options);
  }

  async getById(tenant: Tenant, characterId: string, options?: ServiceOptions): Promise<Character> {
    api.setTenant(tenant);
    return api.getOne<Character>(`${this.basePath}/${characterId}`, options);
  }

  async update(tenant: Tenant, characterId: string, data: UpdateCharacterData): Promise<void> {
    api.setTenant(tenant);
    await api.patch(`${this.basePath}/${characterId}`, {
      data: {
        type: "characters",
        id: characterId,
        attributes: data,
      },
    });
  }
}

export const charactersService = new CharactersService();
```

## BaseService Template Methods

| Method | Purpose |
|--------|---------|
| `getAll<T>(options?)` | Fetch list with query support |
| `getById<T>(id, options?)` | Fetch single by ID |
| `exists(id, options?)` | Check existence (handles 404 → false) |
| `create<T, D>(data, options?)` | POST with validation |
| `update<T, D>(id, data, options?)` | PUT with validation |
| `patch<T, D>(id, data, options?)` | PATCH for partial updates |
| `delete(id, options?)` | DELETE resource |
| `createBatch<T, D>(items, options?, batchOptions?)` | Concurrent creates |
| `updateBatch<T, D>(updates, options?, batchOptions?)` | Concurrent updates |
| `deleteBatch(ids, options?, batchOptions?)` | Concurrent deletes |

## Multi-Tenancy in Services

**Every tenant-scoped method takes `tenant` as first parameter:**

```typescript
async getAllBans(tenant: Tenant, options?: QueryOptions): Promise<Ban[]> {
  api.setTenant(tenant);  // ← Sets tenant headers before request
  // ...
}
```

**Exception:** Tenant-management services (TenantsService) don't take tenant parameter since they manage tenants themselves.

## JSON:API Request Format

All write operations use JSON:API envelope:

```typescript
{
  data: {
    type: "resourceType",
    id: "identifier",        // Required for update/patch
    attributes: { /* data */ }
  }
}
```

## Update Pattern (Immutable)

Merge existing attributes with updates, return new object:

```typescript
async updateTenant(tenant: TenantBasic, updates: Partial<TenantBasicAttributes>): Promise<TenantBasic> {
  const input = {
    data: {
      id: tenant.id,
      type: 'tenants',
      attributes: { ...tenant.attributes, ...updates },
    },
  };
  await this.patch<void, typeof input>(tenant.id, input);
  return { ...tenant, attributes: { ...tenant.attributes, ...updates } };
}
```

## Exports (index.ts)

```typescript
// Base
export { BaseService } from './base.service';
export type { ServiceOptions, QueryOptions, BatchResult, ValidationError } from './base.service';

// Singleton instances
export { bansService } from './bans.service';
export { charactersService } from './characters.service';
export { tenantsService } from './tenants.service';
// ...

// Types re-exported per service
export type { Ban, BanAttributes, CreateBanRequest } from './bans.service';
```
