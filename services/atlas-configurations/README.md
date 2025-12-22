# atlas-configurations
Mushroom game configurations Service

## Overview

A RESTful service which provides configuration management for the Atlas platform. This service allows you to create, retrieve, update, and delete configuration templates, tenants, and service configurations.

## Environment Variables

The following environment variables are required for the service to function properly:

- `JAEGER_HOST_PORT` - Jaeger host and port for distributed tracing
- `LOG_LEVEL` - Logging level (Panic / Fatal / Error / Warn / Info / Debug / Trace)
- `DB_USER` - PostgreSQL database username
- `DB_PASSWORD` - PostgreSQL database password
- `DB_HOST` - PostgreSQL database host
- `DB_PORT` - PostgreSQL database port
- `DB_NAME` - PostgreSQL database name

### Seed Data Configuration

- `SEED_DATA_PATH` - Path to seed data directory (default: `/seed-data`)
- `SEED_ENABLED` - Enable/disable automatic seeding on startup (default: `true`)

## Seed Data

The service supports automatic importing of template configurations from JSON files on startup. This is useful for initializing a fresh database with default configurations.

### How It Works

1. On startup (after database migrations), the seeder scans the seed data directory
2. For each JSON file in `templates/` subdirectory:
   - Extracts `region`, `majorVersion`, and `minorVersion` from the JSON
   - Checks if a template with those identifiers already exists
   - If not found, imports the template
   - If found, skips (existing data is never overwritten)
3. Logs a summary of imported/skipped/failed files

### Directory Structure

```
seed-data/
└── templates/
    ├── template_gms_83_1.json
    ├── template_gms_95_1.json
    └── template_jms_185_1.json
```

### File Naming Convention

Files should be named as: `template_{region}_{majorVersion}_{minorVersion}.json`

Examples:
- `template_gms_83_1.json` - GMS version 83.1 template
- `template_jms_185_1.json` - JMS version 185.1 template

### Adding New Seed Files

1. Create a JSON file with the required structure (must include `region`, `majorVersion`, `minorVersion`)
2. Place it in the `templates/` subdirectory
3. Rebuild the Docker image or mount the directory as a volume

### Disabling Seeding

To disable automatic seeding, set the environment variable:
```
SEED_ENABLED=false
```

## API Endpoints

The service exposes the following RESTful endpoints:

### Configuration Templates

- `GET /api/configurations/templates` - Get all configuration templates
- `GET /api/configurations/templates?region={region}&majorVersion={majorVersion}&minorVersion={minorVersion}` - Get configuration templates by region and version
- `POST /api/configurations/templates` - Create a new configuration template
- `PATCH /api/configurations/templates/{templateId}` - Update an existing configuration template
- `DELETE /api/configurations/templates/{templateId}` - Delete a configuration template

### Configuration Tenants

- `GET /api/configurations/tenants` - Get all configuration tenants
- `GET /api/configurations/tenants/{tenantId}` - Get a specific configuration tenant
- `POST /api/configurations/tenants` - Create a new configuration tenant
- `PATCH /api/configurations/tenants/{tenantId}` - Update a configuration tenant
- `DELETE /api/configurations/tenants/{tenantId}` - Delete a configuration tenant

### Service Configurations

- `GET /api/configurations/services` - Get all service configurations
- `GET /api/configurations/services/{serviceId}` - Get a specific service configuration
