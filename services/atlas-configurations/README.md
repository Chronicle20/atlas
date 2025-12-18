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
