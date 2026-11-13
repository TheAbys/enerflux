#!/bin/sh
set -eu

until pg_isready -h enerflux-postgres -p 5432 -U "$POSTGRES_USER" -d "$POSTGRES_DB"; do
  sleep 2
done

psql -v ON_ERROR_STOP=1 "$DATABASE_URL" <<'SQL'
CREATE TABLE IF NOT EXISTS enerflux_schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
SQL

for migration in /migrations/*.sql; do
  version=$(basename "$migration" .sql)

  if psql -qAt -v ON_ERROR_STOP=1 -v version="$version" "$DATABASE_URL" <<'SQL' | grep -qx 1; then
SELECT 1
FROM enerflux_schema_migrations
WHERE version = :'version';
SQL
    echo "Migration already applied: $version"
    continue
  fi

  echo "Applying migration: $version"
  psql --single-transaction -v ON_ERROR_STOP=1 "$DATABASE_URL" -f "$migration"
  psql -v ON_ERROR_STOP=1 -v version="$version" "$DATABASE_URL" <<'SQL'
INSERT INTO enerflux_schema_migrations (version)
VALUES (:'version')
ON CONFLICT (version) DO NOTHING;
SQL
done
