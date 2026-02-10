---
name: frontend-dev-guidelines
description: Skill for creating and modifying the Atlas UI frontend using Next.js 16, TypeScript, React 19, shadcn/ui, TanStack React Query, react-hook-form with Zod validation, and Tailwind CSS with multi-tenant context.
---


# Frontend Development Skill

## Purpose
Provide a composable entry point that activates when working on the Atlas UI service. This skill aligns development and AI generation with the established frontend architecture patterns and conventions.

## When to Use
Activate when working on:
- Any file inside `services/atlas-ui/`
- React components (`.tsx` files in `components/` or `app/`)
- Custom hooks (`lib/hooks/` or `lib/hooks/api/`)
- API service layer (`services/api/`)
- Zod validation schemas (`lib/schemas/`)
- TypeScript type definitions (`types/`)
- React Query integration and cache management
- Multi-tenancy context and tenant switching
- Form components using react-hook-form
- Data table configurations
- Styling with Tailwind CSS and shadcn/ui
- Testing with Jest and React Testing Library

---

## Quick Start Checklist
- [ ] **Component** follows presentational/container split (ui/ vs features/)
- [ ] **Types** defined with JSON:API structure (`id` + `attributes`)
- [ ] **Service** extends `BaseService` or uses direct API client pattern
- [ ] **Hook** uses query key factory with hierarchical keys (`as const`)
- [ ] **Tenant context** injected via explicit parameter or `useTenant()` hook
- [ ] **Form** uses `react-hook-form` with `zodResolver` and Zod schema
- [ ] **Validation schema** defined in `lib/schemas/` with inferred types
- [ ] **Loading state** uses skeleton components, not spinners (except submit buttons)
- [ ] **Error handling** uses `createErrorFromUnknown()` and toast notifications
- [ ] **Styling** uses Tailwind utility classes with `cn()` helper
- [ ] **Tests** written with Jest + React Testing Library
- [ ] **Test execution** verified before claiming completion

---

## Standard Implementation Workflow

**MANDATORY:** Follow this workflow for ALL code changes.

### Implementation Steps

When modifying any UI code:

1. **Read existing code** before making changes — understand the current patterns in use
2. **Implement changes** following the patterns documented in this skill
3. **Ensure tenant context** is properly handled:
   - Resource-specific hooks: explicit `tenant` parameter
   - Global resource hooks: `useTenant()` context hook
   - Tenant-agnostic operations: no tenant needed
4. **Update types** if API contracts changed (`types/models/`, `types/api/`)
5. **Update service layer** if new API endpoints are needed
6. **Update query hooks** if data fetching patterns changed
7. **Run tests BEFORE claiming completion**:
   ```bash
   npm test
   ```
8. **Fix any failures** — Do NOT skip or ignore test failures
9. **Verify build**:
   ```bash
   npm run build
   ```
10. **Report test results** with actual command output, not assumptions

### Critical Rules

- **Never skip test execution** — Running tests is mandatory, not optional
- **Never assume tests will pass** — Always verify with actual execution
- **Never mutate state directly** — Use immutable update patterns
- **Never bypass tenant context** — All tenant-scoped API calls must include tenant headers
- **Never use `any` type** — TypeScript strict mode is enabled; use proper types
- **Never inline Zod schemas in components** — Define schemas in `lib/schemas/`
- **Always use `cn()` for conditional classes** — Never manual string concatenation
- **Always use skeleton components for loading** — Not raw spinners in content areas
- **Always use toast for user feedback** — `toast.success()`, `toast.error()` via sonner
- **Always verify test output** before marking work complete

### When Tests Fail

If `npm test` reports failures:

1. **Read the error message completely** — Understand what broke
2. **Check for missing mocks** — Most common cause of component test failures
3. **Update mocks to match services** — Add/modify mock implementations
4. **Re-run tests** — Verify the fix didn't break other tests
5. **Do not proceed** until all tests pass

See [Testing Guide](resources/testing-guide.md) for comprehensive testing guidelines.

---

## Key Principles
1. **JSON:API Compliance** — All models use `{ id, attributes }` structure matching backend services.
2. **Multi-Tenancy First** — Every data operation is tenant-scoped via context or explicit parameter.
3. **Type Safety** — TypeScript strict mode with no `any`; use type guards for runtime checks.
4. **Server State via React Query** — All server data managed through TanStack React Query hooks.
5. **Composition over Configuration** — shadcn/ui composable primitives, not monolithic components.
6. **Immutable Updates** — Spread operators for state updates; never mutate props or state directly.

---

## File Responsibilities

| Location | Primary Responsibility |
|----------|------------------------|
| `app/*/page.tsx` | Route pages — data fetching, layout, and composition |
| `app/layout.tsx` | Root layout — providers, sidebar, breadcrumbs |
| `components/ui/` | shadcn/ui base components — buttons, dialogs, inputs |
| `components/common/` | Shared presentational components |
| `components/features/` | Feature-specific container components |
| `components/providers/` | React context provider wrappers |
| `lib/api/client.ts` | Singleton API client with caching, retry, dedup |
| `lib/api/errors.ts` | Error transformation and classification |
| `lib/hooks/api/` | React Query hooks — queries, mutations, invalidation |
| `lib/schemas/` | Zod validation schemas with inferred types |
| `lib/utils.ts` | `cn()` classname utility |
| `services/api/` | Service classes — BaseService + concrete services |
| `types/models/` | Domain model interfaces (JSON:API format) |
| `types/api/` | API response/error type definitions |
| `context/` | React context definitions (tenant, etc.) |

---

## Navigation Guide

| Topic | Reference |
|-------|-----------|
| Architecture Overview | [resources/architecture-overview.md](resources/architecture-overview.md) |
| Service Layer Patterns | [resources/patterns-service-layer.md](resources/patterns-service-layer.md) |
| React Query & Hooks | [resources/patterns-react-query.md](resources/patterns-react-query.md) |
| Component Patterns | [resources/patterns-components.md](resources/patterns-components.md) |
| Routing & Pages | [resources/patterns-routing.md](resources/patterns-routing.md) |
| Forms & Validation | [resources/patterns-forms-validation.md](resources/patterns-forms-validation.md) |
| Styling & Theming | [resources/patterns-styling.md](resources/patterns-styling.md) |
| API Client | [resources/patterns-api-client.md](resources/patterns-api-client.md) |
| Type System | [resources/patterns-types.md](resources/patterns-types.md) |
| Multi-Tenancy | [resources/patterns-multitenancy.md](resources/patterns-multitenancy.md) |
| Testing Guide | [resources/testing-guide.md](resources/testing-guide.md) |
| Anti-Patterns | [resources/anti-patterns.md](resources/anti-patterns.md) |
| AI Code Guidance | [resources/ai-guidance.md](resources/ai-guidance.md) |

---
