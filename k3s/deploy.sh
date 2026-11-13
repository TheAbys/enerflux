#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "$SCRIPT_DIR/.." && pwd)"

ENV_FILE="${ENV_FILE:-$PROJECT_ROOT/.env}"
NAMESPACE="${NAMESPACE:-enerflux}"
IMAGE_PREFIX="${IMAGE_PREFIX:-ghcr.io/theabys}"
IMAGE_PREFIX="${IMAGE_PREFIX%/}"
IMAGE_TAG="${IMAGE_TAG:-develop}"
IMAGE_PULL_POLICY="${IMAGE_PULL_POLICY:-Always}"
API_NODE_PORT="${API_NODE_PORT:-30080}"
ROLLOUT_TIMEOUT="${ROLLOUT_TIMEOUT:-180s}"

for command in kubectl envsubst python3; do
  if ! command -v "$command" >/dev/null 2>&1; then
    echo "$command is required to deploy EnerFlux." >&2
    exit 1
  fi
done

if [[ ! -f "$ENV_FILE" ]]; then
  echo "Environment file not found: $ENV_FILE" >&2
  echo "Set ENV_FILE to the application's .env file." >&2
  exit 1
fi

runtime_env="$(mktemp)"
chmod 600 "$runtime_env"
trap 'rm -f "$runtime_env"' EXIT

python3 - "$ENV_FILE" "$runtime_env" <<'PY'
import sys
from urllib.parse import parse_qsl, urlencode, unquote, urlsplit, urlunsplit

source, target = sys.argv[1:]
values = {}
with open(source, encoding="utf-8") as env_file:
    for raw_line in env_file:
        line = raw_line.strip()
        if not line or line.startswith("#"):
            continue
        key, separator, value = line.partition("=")
        if not separator:
            continue
        key = key.strip()
        value = value.strip()
        if len(value) >= 2 and value[0] == value[-1] and value[0] in "\"'":
            value = value[1:-1]
        values[key] = value

database_url = values.get("DATABASE_URL", "")
parts = urlsplit(database_url)
if parts.scheme not in ("postgres", "postgresql") or not parts.username or not parts.password:
    raise SystemExit("DATABASE_URL must be a PostgreSQL URL with a username and password.")

username = unquote(parts.username)
password = unquote(parts.password)
database = unquote(parts.path.lstrip("/"))
if not database:
    raise SystemExit("DATABASE_URL must include a database name.")
if any("\n" in value or "\r" in value for value in (username, password, database)):
    raise SystemExit("PostgreSQL credentials and database name cannot contain newlines.")

query = [(key, value) for key, value in parse_qsl(parts.query, keep_blank_values=True) if key != "sslmode"]
query.append(("sslmode", "disable"))
userinfo = parts.netloc.rsplit("@", 1)[0]
in_cluster_url = urlunsplit((parts.scheme, f"{userinfo}@enerflux-postgres:5432", parts.path, urlencode(query), ""))

entries = {
    "POSTGRES_USER": username,
    "POSTGRES_PASSWORD": password,
    "POSTGRES_DB": database,
    "DATABASE_URL": in_cluster_url,
}
with open(target, "w", encoding="utf-8") as output:
    for key, value in entries.items():
        if "\n" in value or "\r" in value:
            raise SystemExit(f"{key} cannot contain newlines.")
        output.write(f"{key}={value}\n")
PY

API_IMAGE="$IMAGE_PREFIX/enerflux-api:$IMAGE_TAG"
AGENT_IMAGE="$IMAGE_PREFIX/enerflux-agent:$IMAGE_TAG"
export API_IMAGE AGENT_IMAGE IMAGE_PULL_POLICY API_NODE_PORT

kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -

kubectl create secret generic enerflux-app-config \
  --namespace "$NAMESPACE" \
  --from-env-file="$ENV_FILE" \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl create secret generic enerflux-db-auth \
  --namespace "$NAMESPACE" \
  --from-env-file="$runtime_env" \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl create configmap enerflux-stack-config \
  --namespace "$NAMESPACE" \
  --from-file=mosquitto.conf="$PROJECT_ROOT/docker/mosquitto/mosquitto.conf" \
  --from-file=migrate.sh="$SCRIPT_DIR/migrate.sh" \
  --from-file="$PROJECT_ROOT/migrations" \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl apply --namespace "$NAMESPACE" -f "$SCRIPT_DIR/infrastructure.yaml"
kubectl rollout status statefulset/enerflux-postgres \
  --namespace "$NAMESPACE" --timeout="$ROLLOUT_TIMEOUT"
kubectl rollout status statefulset/enerflux-mqtt \
  --namespace "$NAMESPACE" --timeout="$ROLLOUT_TIMEOUT"

job_name="$(kubectl create --namespace "$NAMESPACE" -f "$SCRIPT_DIR/migration-job.yaml" -o name)"
if ! kubectl wait --namespace "$NAMESPACE" --for=condition=complete \
  --timeout="$ROLLOUT_TIMEOUT" "$job_name"; then
  kubectl logs --namespace "$NAMESPACE" "$job_name" --all-containers=true || true
  exit 1
fi

envsubst '${API_IMAGE} ${AGENT_IMAGE} ${IMAGE_PULL_POLICY} ${API_NODE_PORT}' \
  < "$SCRIPT_DIR/application.yaml" \
  | kubectl apply --namespace "$NAMESPACE" -f -

kubectl rollout status deployment/enerflux-api \
  --namespace "$NAMESPACE" --timeout="$ROLLOUT_TIMEOUT"
kubectl rollout status deployment/enerflux-agent \
  --namespace "$NAMESPACE" --timeout="$ROLLOUT_TIMEOUT"

echo "EnerFlux is deployed in namespace '$NAMESPACE'."
echo "API: http://<k3s-node-ip>:$API_NODE_PORT"
echo "MQTT for LAN devices: <k3s-node-ip>:31883"
