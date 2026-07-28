# Storage

This service uses Redis as a tenant-scoped cache via `atlas.TenantRegistry`. It also maintains a non-persistent, in-process cache of equipment requirement data (`ReqLevel`, `ReqJob`, `ReqStr`, `ReqDex`, `ReqInt`, `ReqLuk`) fetched from atlas-data, keyed by tenant id and template id; this cache is local to each service instance and is not backed by Redis or a database. No relational database is used.

## Tables

None.

## Relationships

None.

## Indexes

None.

## Migration Rules

Not applicable. All state is cached in Redis under the `effective-stats` namespace, keyed by character ID. State is rebuilt from external services on demand during lazy initialization. The in-process equipment-requirements cache is lost on service restart and is repopulated on demand from atlas-data.
