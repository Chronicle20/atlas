# Storage

This service uses Redis as a tenant-scoped cache via `atlas.TenantRegistry`. No relational database is used.

## Tables

None.

## Relationships

None.

## Indexes

None.

## Migration Rules

Not applicable. All state is cached in Redis under the `effective-stats` namespace, keyed by character ID. State is rebuilt from external services on demand during lazy initialization.
