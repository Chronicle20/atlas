-- One-shot operator script. Runs after atlas-maps deploy with the new
-- character_locations table, before atlas-character drops map_id/instance.
-- Re-run is idempotent (PK collision = upsert via ON CONFLICT).
INSERT INTO character_locations
  (tenant_id, character_id, world_id, channel_id, map_id, instance, updated_at)
SELECT
  tenant_id,
  id,
  world,
  0,
  map_id,
  instance,
  NOW()
FROM characters
ON CONFLICT (tenant_id, character_id) DO UPDATE
  SET map_id = EXCLUDED.map_id,
      instance = EXCLUDED.instance,
      updated_at = NOW();
