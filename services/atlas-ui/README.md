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
- **State Management**: Local state → React Context → Global store
- **Data Fetching**: React Query for server state management
- **Styling**: Tailwind CSS with shadcn/ui component system

### API Integration

The UI integrates with Atlas microservices using typed API clients:

- Type-safe responses with `ApiResponse<T>` wrapper
- Proper error handling with `ApiError` types
- Multi-tenant headers (`TENANT_ID`, `REGION`, etc.)

For detailed architectural guidelines, see `CLAUDE.md`.
