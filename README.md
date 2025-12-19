# atlas
Atlas Mushroom Game Service

[![PR Validation](https://github.com/Chronicle20/atlas/actions/workflows/pr-validation.yml/badge.svg)](https://github.com/Chronicle20/atlas/actions/workflows/pr-validation.yml)
[![Main - Build and Publish](https://github.com/Chronicle20/atlas/actions/workflows/main-publish.yml/badge.svg)](https://github.com/Chronicle20/atlas/actions/workflows/main-publish.yml)

## Project Structure

```
atlas/
├── services/           # Microservices (44 total)
│   ├── atlas-account/
│   ├── atlas-login/
│   ├── atlas-ui/       # Next.js frontend
│   └── ...
├── libs/               # Shared libraries (6 total)
│   ├── atlas-model/
│   ├── atlas-kafka/
│   ├── atlas-rest/
│   └── ...
├── tools/              # Build scripts
├── .github/            # CI/CD workflows
└── go.work             # Go workspace
```

## CI/CD

This monorepo uses GitHub Actions for continuous integration and deployment. The CI/CD system automatically detects which services have changed and only builds/tests those services.

### Workflows

#### PR Validation (`pr-validation.yml`)

Runs on every pull request to `main`:

1. **Change Detection** - Identifies modified services and libraries
2. **Go Library Tests** - Tests changed libraries with race detection and coverage (75% threshold for atlas-model)
3. **Go Service Tests** - Tests changed services
4. **UI Tests** - Tests atlas-ui if changed (Node.js/React)
5. **Docker Build Validation** - Builds Docker images without pushing

#### Main Publish (`main-publish.yml`)

Runs on push to `main`:

1. **Change Detection** - Identifies modified services
2. **Multi-Arch Builds** - Builds Docker images for AMD64 and ARM64
3. **Manifest Creation** - Creates multi-architecture manifests
4. **Registry Push** - Pushes to GitHub Container Registry

### Manual Triggers

Both workflows support manual triggering via `workflow_dispatch`:

```bash
# Force validation of all services
gh workflow run pr-validation.yml --field force-all=true

# Build and publish specific service
gh workflow run main-publish.yml --field service=atlas-account

# Build and publish all services
gh workflow run main-publish.yml --field force-all=true
```

### Service Configuration

Services are configured in `.github/config/services.json`. This file defines:
- Service names and paths
- Go module paths
- Docker image names
- Coverage thresholds (for libraries)

### Docker Images

Images are published to GitHub Container Registry:
```
ghcr.io/chronicle20/{service-name}/{service-name}:latest
```

Multi-architecture support:
- `linux/amd64`
- `linux/arm64`

## Development

### Prerequisites

- Go 1.25.5+
- Node.js 22+ (for atlas-ui)
- Docker

### Building Locally

```bash
# Test all Go modules
./tools/test-all-go.sh

# Build all Docker images
./tools/build-services.sh

# Tidy all Go modules
./tools/tidy-all-go.sh
```

### Go Workspace

The monorepo uses Go workspaces (`go.work`) to manage multiple modules. All services and libraries are included in the workspace.

## Services

| Service | Description |
|---------|-------------|
| atlas-account | Account management |
| atlas-login | Login server |
| atlas-channel | Channel server |
| atlas-world | World server |
| atlas-character | Character service |
| atlas-ui | Web frontend (Next.js) |
| ... | See `services/` directory |

## Libraries

| Library | Description |
|---------|-------------|
| atlas-model | Domain models and utilities |
| atlas-kafka | Kafka messaging integration |
| atlas-rest | REST API utilities |
| atlas-tenant | Multi-tenancy support |
| atlas-socket | Socket communication |
| atlas-constants | Shared constants |
