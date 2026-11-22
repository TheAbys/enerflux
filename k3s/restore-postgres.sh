#!/usr/bin/env bash
set -Eeuo pipefail

NAMESPACE="${NAMESPACE:-enerflux}"
POD_READY_TIMEOUT="${POD_READY_TIMEOUT:-120s}"
BACKUP_FILE="${BACKUP_FILE:-}"

if ! command -v kubectl >/dev/null 2>&1; then
  echo "kubectl is required to restore EnerFlux PostgreSQL." >&2
  exit 1
fi
if [[ -z "$BACKUP_FILE" || ! -f "$BACKUP_FILE" ]]; then
  echo "Set BACKUP_FILE to a PostgreSQL dump created by backup-postgres.sh." >&2
  exit 1
fi
if [[ ! -s "$BACKUP_FILE" ]]; then
  echo "Backup file is empty: $BACKUP_FILE" >&2
  exit 1
fi
if [[ "${CONFIRM_RESTORE:-}" != "yes" ]]; then
  echo "Restore replaces database objects. Stop the API and agent, then rerun with CONFIRM_RESTORE=yes." >&2
  exit 1
fi

for deployment in enerflux-api enerflux-agent; do
  replicas="$(kubectl get deployment "$deployment" --namespace "$NAMESPACE" --output 'jsonpath={.spec.replicas}')"
  if [[ "$replicas" != "0" ]]; then
    echo "Scale deployment '$deployment' to zero before restoring the database." >&2
    exit 1
  fi
done

pod_name="$(kubectl get pods --namespace "$NAMESPACE" \
  --selector app.kubernetes.io/name=enerflux-postgres \
  --output 'jsonpath={.items[0].metadata.name}')"
if [[ -z "$pod_name" ]]; then
  echo "No PostgreSQL pod found in namespace '$NAMESPACE'." >&2
  exit 1
fi

kubectl wait --namespace "$NAMESPACE" --for=condition=Ready \
  --timeout="$POD_READY_TIMEOUT" "pod/$pod_name" >/dev/null

kubectl exec --namespace "$NAMESPACE" --stdin "$pod_name" -- sh -ec \
  'pg_restore --clean --if-exists --no-owner --no-privileges --exit-on-error -U "$POSTGRES_USER" -d "$POSTGRES_DB"' \
  < "$BACKUP_FILE"

echo "PostgreSQL restore completed from $BACKUP_FILE"
