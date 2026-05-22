# Runbook: Remove stale `layer-*.png` files from MinIO

## When to use

Task-071 moved per-map layer composition to render-time (atlas-renders)
and stopped emitting `layer-N.png` files from atlas-data during ingest.
Pre-refactor uploads in any (tenant, region, version) tuple are now dead
weight in MinIO. Each cleanup is one-shot per env.

## What to clean

Under `atlas-assets`, every prefix matching:

```
tenants/<tenantId>/regions/<region>/versions/<x.y>/map/<mapId>/layers/
```

(Note: the shared scope prefix `shared/regions/.../layers/` is also fair
game if your env populated it from a pre-refactor ingest.)

## Procedure (per env)

```bash
# Enumerate the (tenant, region, version) tuples currently restored.
mc alias set adm http://minio.minio.svc.cluster.local:9000 <accessKey> <secretKey>
mc find adm/atlas-assets --regex 'layers/$' --type d | head -20

# Dry-run.
mc find adm/atlas-assets --regex 'layers/' --type f | head -20

# Execute.
mc find adm/atlas-assets --regex 'layers/' --type f -exec 'mc rm {}'
```

For atlas-main run the execute step inside a one-shot Job (image
`minio/mc:latest`) using the existing MinIO credentials secret.

## Verification

```bash
mc find adm/atlas-assets --regex 'layers/' --type f | wc -l   # expect 0
```

## Frequency

Run once per env after task-076 lands. New `layers/` prefixes should not
reappear — atlas-renders composites in-memory now.
