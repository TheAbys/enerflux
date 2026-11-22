#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "$SCRIPT_DIR/.." && pwd)"

NAMESPACE="${NAMESPACE:-enerflux}"
BACKUP_DIR="${BACKUP_DIR:-$PROJECT_ROOT/backups}"
POD_READY_TIMEOUT="${POD_READY_TIMEOUT:-120s}"
BACKUP_TIMESTAMP="${BACKUP_TIMESTAMP:-$(date -u +%Y%m%dT%H%M%SZ)}"

if ! command -v kubectl >/dev/null 2>&1; then
  echo "kubectl is required to back up EnerFlux PostgreSQL." >&2
  exit 1
fi
if [[ ! "$BACKUP_TIMESTAMP" =~ ^[A-Za-z0-9._-]+$ ]]; then
  echo "BACKUP_TIMESTAMP may contain only letters, numbers, dots, underscores, and hyphens." >&2
  exit 1
fi

pod_name="$(kubectl get pods --namespace "$NAMESPACE" \
  --selector app.kubernetes.io/name=enerflux-postgres \
  --output 'jsonpath={.items[0].metadata.name}')"
if [[ -z "$pod_name" ]]; then
  echo "No PostgreSQL pod found in namespace '$NAMESPACE'." >&2
  exit 1
fi

kubectl wait --namespace "$NAMESPACE" --for=condition=Ready \
  --timeout="$POD_READY_TIMEOUT" "pod/$pod_name" >/dev/null

umask 077
mkdir -p "$BACKUP_DIR"
backup_file="$BACKUP_DIR/enerflux-postgres-$BACKUP_TIMESTAMP.dump"
temporary_file="$backup_file.partial"
if [[ -e "$backup_file" ]]; then
  echo "Backup already exists: $backup_file" >&2
  exit 1
fi
trap 'rm -f "$temporary_file"' EXIT

kubectl exec --namespace "$NAMESPACE" "$pod_name" -- sh -ec \
  'pg_dump --format=custom --no-owner --no-privileges -U "$POSTGRES_USER" "$POSTGRES_DB"' \
  > "$temporary_file"

if [[ ! -s "$temporary_file" ]]; then
  echo "PostgreSQL backup was empty; refusing to keep it." >&2
  exit 1
fi

mv "$temporary_file" "$backup_file"
trap - EXIT
echo "PostgreSQL backup saved to $backup_file"
