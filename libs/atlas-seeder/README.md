# atlas-seeder

Shared library for tenant-scoped catalog seeding. Provides:

- `Subdomain[J, M]` generic interface for one catalog dataset.
- `Group` for one `(POST /<prefix>/seed, GET /<prefix>/seed/status)` endpoint pair.
- `CatalogSource` abstraction over file lookup (v1: filesystem rooted at `SEED_CATALOG_ROOT`).
- `RegisterRoutes` to wire HTTP handlers.
- `SeedState` GORM entity persisting `(tenant_id, group_name) -> catalog_revision` per service.

See `docs/tasks/task-072-shared-seeder-catalog/design.md` for architecture.
