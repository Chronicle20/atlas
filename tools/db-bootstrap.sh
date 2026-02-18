#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<EOF
Usage: $(basename "$0") [OPTIONS]

Bootstrap PostgreSQL databases for Atlas services.
Creates the app role, databases, and installs required extensions.

Options:
  -H HOST      Postgres host            (default: 127.0.0.1,    env: PGHOST)
  -p PORT      Postgres port            (default: 5432,         env: PGPORT)
  -U USER      Postgres admin user      (default: postgres,     env: PGADMIN_USER)
  -W PASSWORD  Postgres admin password  (optional,              env: PGPASSWORD)
  -d DB        Postgres admin database  (default: postgres,     env: PGADMIN_DB)
  -u USER      Application role name    (default: atlas_app,    env: APP_USER)
  -w PASSWORD  Application role password (required,             env: APP_PASSWORD)
  -h           Show this help message

Examples:
  $(basename "$0") -w secret
  $(basename "$0") -H db.local -p 5433 -U admin -W adminpw -w secret
  APP_PASSWORD=secret $(basename "$0")
EOF
}

# ---- Defaults (env fallback) ----
PGHOST="${PGHOST:-127.0.0.1}"
PGPORT="${PGPORT:-5432}"
PGADMIN_USER="${PGADMIN_USER:-postgres}"
PGADMIN_DB="${PGADMIN_DB:-postgres}"
APP_USER="${APP_USER:-atlas_app}"
APP_PASSWORD="${APP_PASSWORD:-}"

while getopts ":H:p:U:W:d:u:w:h" opt; do
  case "$opt" in
    H) PGHOST="$OPTARG" ;;
    p) PGPORT="$OPTARG" ;;
    U) PGADMIN_USER="$OPTARG" ;;
    W) export PGPASSWORD="$OPTARG" ;;
    d) PGADMIN_DB="$OPTARG" ;;
    u) APP_USER="$OPTARG" ;;
    w) APP_PASSWORD="$OPTARG" ;;
    h) usage; exit 0 ;;
    :) echo "Error: -${OPTARG} requires an argument." >&2; usage; exit 1 ;;
    *) echo "Error: unknown option -${OPTARG}" >&2; usage; exit 1 ;;
  esac
done
shift $((OPTIND - 1))

if [[ -z "$APP_PASSWORD" ]]; then
  echo "Error: APP_PASSWORD is required. Set via -w or APP_PASSWORD env var." >&2
  echo >&2
  usage >&2
  exit 1
fi


# ---- Databases (edit this list for now) ----
DBS=(
  atlas-accounts
  atlas-ban
  atlas-buddies
  atlas-cashshop
  atlas-characters
  atlas-configurations
  atlas-data
  atlas-drops
  atlas-families
  atlas-fame
  atlas-gachapons
  atlas-guilds
  atlas-inventory
  atlas-keys
  atlas-map-actions
  atlas-maps
  atlas-marriages
  atlas-notes
  atlas-npc-conversations
  atlas-npc-shops
  atlas-party-quests
  atlas-pets
  atlas-portal-actions
  atlas-quest
  atlas-reactor-actions
  atlas-skills
  atlas-storage
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


