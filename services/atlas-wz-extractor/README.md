# atlas-wz-extractor

Parses proprietary WZ binary archive files and extracts their contents into two output formats: HaRepacker-compatible XML files and PNG icon images. The service auto-detects game version and encryption type (GMS, KMS, or unencrypted) via brute-force version matching.

Extraction is tenant-aware. Output is organized by tenant ID, region, and game version. The service exposes a single REST endpoint that triggers asynchronous extraction and returns immediately.

## Usage Guide

### 1. Place WZ Files

Copy the `.wz` files you want to extract into the input directory. In Kubernetes this is the NFS volume mounted at `/usr/wz-input` (backed by `atlas-wz-input-pvc`). For local development the path is set by the `INPUT_WZ_DIR` environment variable.

```
<INPUT_WZ_DIR>/
├── Character.wz
├── Item.wz
├── Map.wz
├── Mob.wz
├── Npc.wz
├── Reactor.wz
├── Skill.wz
├── String.wz
└── ...
```

All `*.wz` files in this directory will be processed. You may include any combination of WZ files -- the service processes whatever it finds and skips files it cannot parse.

### 2. Trigger Extraction

Send a `POST` request to the extraction endpoint with the required tenant headers.

**Endpoint:** `POST /api/wz/extractions`

**Required Headers:**

| Header | Type | Description |
|---|---|---|
| `TENANT_ID` | UUID | Tenant identifier (e.g. `4ec40a5a-e596-4613-b498-e42450505e91`) |
| `REGION` | string | Game region (e.g. `GMS`, `KMS`) |
| `MAJOR_VERSION` | integer | Game major version (e.g. `83`) |
| `MINOR_VERSION` | integer | Game minor version (e.g. `1`) |

**Optional Query Parameters:**

| Parameter | Value | Description |
|---|---|---|
| `xmlOnly` | `true` | Only perform XML serialization (skip icon extraction) |
| `imagesOnly` | `true` | Only perform icon extraction (skip XML serialization) |

**Example -- full extraction (XML + icons):**

```bash
curl -X POST http://localhost:8083/api/wz/extractions \
  -H "TENANT_ID: 4ec40a5a-e596-4613-b498-e42450505e91" \
  -H "REGION: GMS" \
  -H "MAJOR_VERSION: 83" \
  -H "MINOR_VERSION: 1"
```

**Example -- XML only:**

```bash
curl -X POST "http://localhost:8083/api/wz/extractions?xmlOnly=true" \
  -H "TENANT_ID: 4ec40a5a-e596-4613-b498-e42450505e91" \
  -H "REGION: GMS" \
  -H "MAJOR_VERSION: 83" \
  -H "MINOR_VERSION: 1"
```

**Example -- icons only:**

```bash
curl -X POST "http://localhost:8083/api/wz/extractions?imagesOnly=true" \
  -H "TENANT_ID: 4ec40a5a-e596-4613-b498-e42450505e91" \
  -H "REGION: GMS" \
  -H "MAJOR_VERSION: 83" \
  -H "MINOR_VERSION: 1"
```

**Via the ingress (Kubernetes):**

```bash
curl -X POST https://<ingress-host>/api/wz/extractions \
  -H "TENANT_ID: 4ec40a5a-e596-4613-b498-e42450505e91" \
  -H "REGION: GMS" \
  -H "MAJOR_VERSION: 83" \
  -H "MINOR_VERSION: 1"
```

The endpoint returns `202 Accepted` immediately with `{"status": "started"}`. Extraction runs asynchronously in the background. Monitor service logs for progress and completion.

### 3. Output Directory Structure

After extraction completes, output is organized under the tenant's path:

**XML output** (`OUTPUT_XML_DIR`):

```
<OUTPUT_XML_DIR>/
└── <tenantId>/
    └── <region>/
        └── <majorVersion>.<minorVersion>/
            ├── Character.wz/
            │   └── .../<imageName>.img.xml
            ├── Item.wz/
            │   └── .../<imageName>.img.xml
            ├── Map.wz/
            │   └── .../<imageName>.img.xml
            ├── Mob.wz/
            │   └── .../<imageName>.img.xml
            ├── Npc.wz/
            │   └── .../<imageName>.img.xml
            └── ...
```

Each `.img` entry in the WZ archive becomes a separate `.img.xml` file, preserving the full directory hierarchy from the WZ file. The XML format is HaRepacker-compatible.

**Icon output** (`OUTPUT_IMG_DIR`):

```
<OUTPUT_IMG_DIR>/
└── <tenantId>/
    └── <region>/
        └── <majorVersion>.<minorVersion>/
            ├── npc/
            │   └── <npcId>/icon.png
            ├── mob/
            │   └── <mobId>/icon.png
            ├── reactor/
            │   └── <reactorId>/icon.png
            ├── item/
            │   └── <itemId>/icon.png
            └── skill/
                └── <skillId>/icon.png
```

### 4. How Downstream Services Use the Output

- **atlas-data** reads the XML output to populate its PostgreSQL database with static game data. The `atlas-data-pvc` volume is shared between both services.
- **atlas-assets** serves the extracted PNG icons. The `atlas-assets-pvc` volume holds the icon output.

### 5. Typical Workflow

1. Obtain WZ files for the target game version.
2. Copy them to the input volume (`atlas-wz-input-pvc` or local `INPUT_WZ_DIR`).
3. Call the extraction endpoint with the tenant headers for your tenant/region/version.
4. Wait for extraction to complete (monitor logs).
5. Trigger atlas-data to process the newly extracted XML files.

## External Dependencies

- **Jaeger** (optional) -- OpenTelemetry trace export via OTLP/gRPC

## Runtime Configuration

| Variable | Required | Description |
|---|---|---|
| `INPUT_WZ_DIR` | Yes | Path to directory containing `.wz` files |
| `OUTPUT_XML_DIR` | Yes | Root directory for XML output |
| `OUTPUT_IMG_DIR` | Yes | Root directory for PNG icon output |
| `REST_PORT` | No | HTTP listen port (default: `8083`) |
| `LOG_LEVEL` | No | Log level (default: `debug`) |
| `JAEGER_HOST_PORT` | No | OTLP/gRPC endpoint for trace export |

## Kubernetes Storage

| PVC | Mount | Purpose |
|---|---|---|
| `atlas-wz-input-pvc` | `/usr/wz-input` | Input WZ files (NFS) |
| `atlas-data-pvc` | `/usr/data` | XML output (shared with atlas-data) |
| `atlas-assets-pvc` | `/usr/assets` | PNG icon output |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
