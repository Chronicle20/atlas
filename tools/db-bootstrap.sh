#!/usr/bin/env bash
set -euo pipefail


# ---- Connection (override via env if needed) ----
PGHOST="${PGHOST:-127.0.0.1}"
PGPORT="${PGPORT:-5432}"
PGADMIN_USER="${PGADMIN_USER:-postgres}"
PGADMIN_DB="${PGADMIN_DB:-postgres}"

# ---- App user (required) ----
APP_USER="${APP_USER:-atlas_app}"
APP_PASSWORD="${APP_PASSWORD:?set APP_PASSWORD}"


# ---- Databases (edit this list for now) ----
DBS=(
  atlas-accounts
  atlas-buddies
  atlas-cashshop
  atlas-characters
  atlas-configurations
  atlas-data
  atlas-drops
  atlas-equipables
  atlas-fame
  atlas-guilds
  atlas-inventory
  atlas-keys
  atlas-notes
  atlas-npc-conversations
  atlas-npc-shops
  atlas-pets
  atlas-skills
  atlas-tenants
)

PSQL=(
  psql
  -X
  -q
  -v ON_ERROR_STOP=1
  -v VERBOSITY=terse
  -P pager=off
)

ADMIN_URL="postgresql://${PGADMIN_USER}@${PGHOST}:${PGPORT}/${PGADMIN_DB}"

echo "== Target Postgres: ${PGHOST}:${PGPORT} (admin db=${PGADMIN_DB}, user=${PGADMIN_USER})"
echo "== App role: ${APP_USER}"
echo "== Databases: ${DBS[*]}"

# 1) Create/update role
echo "== Creating/updating role ${APP_USER}"
"${PSQL[@]}" "$ADMIN_URL" <<SQL
DO \$\$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '${APP_USER}') THEN
    CREATE ROLE ${APP_USER} LOGIN PASSWORD '${APP_PASSWORD}';
  ELSE
    ALTER ROLE ${APP_USER} LOGIN PASSWORD '${APP_PASSWORD}';
  END IF;
END \$\$;
SQL


# 2) Always install uuid-ossp into template1 so future DBs inherit it
echo '== Installing uuid-ossp into template1 (future DBs inherit this)'
"${PSQL[@]}" "postgresql://${PGADMIN_USER}@${PGHOST}:${PGPORT}/template1" <<'SQL'
SET client_min_messages = warning;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
SQL

# 3) Create DBs (if missing)
echo "== Creating databases if missing (owner=${APP_USER})"
for db in "${DBS[@]}"; do
  # check existence (quiet)
  exists="$("${PSQL[@]}" "$ADMIN_URL" -tAc \
    "SELECT 1 FROM pg_database WHERE datname = '$db'")"

  if [[ "$exists" != "1" ]]; then

    echo "== Creating database: $db"
    createdb -h "$PGHOST" -p "$PGPORT" -U "$PGADMIN_USER" -O "$APP_USER" -- "$db"
  fi

done


# 4) Ensure uuid-ossp exists in each target DB (covers DBs created before template1 change)
echo '== Ensuring uuid-ossp exists in each target DB'
for db in "${DBS[@]}"; do
"${PSQL[@]}" "postgresql://${PGADMIN_USER}@${PGHOST}:${PGPORT}/${db}" <<'SQL'
SET client_min_messages = warning;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
SQL
done


echo "== Done."


