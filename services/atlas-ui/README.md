# Atlas UI

Mushroom game Administrative Web UI built with Next.js, TypeScript, Tailwind CSS, and shadcn/ui.

## Features

- 🎯 **Type Safety First**: Full TypeScript strict mode with comprehensive type definitions
- 🎨 **Modern UI**: Built with shadcn/ui components and Tailwind CSS
- ⚡ **Performance**: Next.js App Router with optimized builds
- 🔒 **Multi-tenant**: Support for multiple game tenants/regions
- 📊 **Admin Tools**: Character, guild, NPC, and game world management

## Development

### Prerequisites

- Node.js 18+
- npm or yarn

### Getting Started

```bash
# Install dependencies
npm install

# Start development server
npm run dev

# Build for production
npm run build

# Run linting
npm run lint
```

The application will be available at `http://localhost:3000`.

### Type Safety

This project uses TypeScript in strict mode with enhanced type checking:

- **Strict Mode**: All TypeScript strict options enabled
- **Type Definitions**: Centralized types in `types/` directory
  - `types/api/` - API response and error types
  - `types/models/` - Domain model types (Tenant, Character, Guild, etc.)
  - `types/components/` - Component prop types
- **Error Handling**: Typed error handling with proper unknown typing

### Architecture

The codebase follows the architectural patterns defined in `CLAUDE.md`:

- **Component Structure**: Organized by scope (ui, common, features)
- **Reusable Components**: Common UI patterns extracted into reusable components (see `docs/reusable-components.md`)
- **Error Handling**: Comprehensive error handling system with user-friendly messages and recovery options (see `docs/error-handling.md`)
- **State Management**: Local state → React Context → Global store
- **Data Fetching**: React Query for server state management
- **Styling**: Tailwind CSS with shadcn/ui component system

### API Integration

The UI integrates with Atlas microservices through a centralized service layer:

**Service Layer (`/services/api/`)**
- Centralized service classes for all domain operations
- Consistent CRUD operations with BaseService architecture
- Request cancellation, deduplication, and caching support
- Automatic error transformation and retry logic

**Available Services:**
- `tenantsService` - Tenant management and configuration
- `accountsService` - Account operations and queries
- `charactersService` - Character management
- `inventoryService` - Inventory and compartment operations
- `mapsService` - Map data and management
- `guildsService` - Guild operations
- `npcsService` - NPC shop and commodity management
- `conversationsService` - NPC conversation management
- `templatesService` - Template management

**Features:**
- Type-safe responses with comprehensive TypeScript support
- Centralized error handling with user-friendly messages
- Multi-tenant headers (`TENANT_ID`, `REGION`, etc.) automatically injected
- React Query integration for caching and state management
- Request deduplication and progress tracking

For detailed architectural guidelines, see `CLAUDE.md`.

### Service Layer Usage

```typescript
// Import services from centralized location
import { tenantsService, accountsService } from '@/services/api';

// Basic CRUD operations
const tenant = await tenantsService.getById('tenant-id');
const accounts = await accountsService.getAll();
const newAccount = await accountsService.create(accountData);

// Advanced operations with options
const charactersPage = await charactersService.getAll({
  page: 1,
  pageSize: 20,
  filters: { accountId: 'account-123' }
});

// Batch operations
const results = await accountsService.batchCreate([
  { name: 'Account 1', /* ... */ },
  { name: 'Account 2', /* ... */ }
]);

// With React Query integration
const { data, isLoading, error } = useQuery({
  queryKey: ['tenants'],
  queryFn: () => tenantsService.getAll()
});
```

## Documentation

- [`CLAUDE.md`](./CLAUDE.md) - Complete architectural guidelines and conventions
- [`docs/service-layer.md`](./docs/service-layer.md) - Centralized API service layer architecture guide
- [`docs/reusable-components.md`](./docs/reusable-components.md) - Guide to reusable UI components
- [`docs/error-handling.md`](./docs/error-handling.md) - Comprehensive error handling patterns and examples
