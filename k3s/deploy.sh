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
INFRA_IMAGE_PULL_POLICY="${INFRA_IMAGE_PULL_POLICY:-IfNotPresent}"
POSTGRES_IMAGE="${POSTGRES_IMAGE:-postgres:16.16}"
MQTT_IMAGE="${MQTT_IMAGE:-eclipse-mosquitto:2.1.2-alpine}"
API_NODE_PORT="${API_NODE_PORT:-30080}"
MQTT_NODE_PORT="${MQTT_NODE_PORT:-31883}"
ROLLOUT_TIMEOUT="${ROLLOUT_TIMEOUT:-660s}"
DEPLOY_REVISION="${DEPLOY_REVISION:-$(date -u +%Y%m%dT%H%M%SZ)-$$}"
INFRA_RELEASE="${INFRA_RELEASE:-enerflux-infrastructure}"
APP_RELEASE="${APP_RELEASE:-enerflux}"
HELM_VALUES_FILE="${HELM_VALUES_FILE:-}"
IMAGE_PULL_SECRET_NAME="${IMAGE_PULL_SECRET_NAME:-}"

for command in kubectl helm python3; do
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
export API_IMAGE AGENT_IMAGE IMAGE_PULL_POLICY API_NODE_PORT DEPLOY_REVISION

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

adopt_resource() {
  local kind="$1"
  local name="$2"
  local release="$3"
  local resource="$kind/$name"
  local owner current_namespace managed_by

  if ! kubectl get "$resource" --namespace "$NAMESPACE" >/dev/null 2>&1; then
    return
  fi

  owner="$(kubectl get "$resource" --namespace "$NAMESPACE" \
    --output 'jsonpath={.metadata.annotations.meta\.helm\.sh/release-name}')"
  current_namespace="$(kubectl get "$resource" --namespace "$NAMESPACE" \
    --output 'jsonpath={.metadata.annotations.meta\.helm\.sh/release-namespace}')"
  managed_by="$(kubectl get "$resource" --namespace "$NAMESPACE" \
    --output 'jsonpath={.metadata.labels.app\.kubernetes\.io/managed-by}')"

  if [[ -n "$owner" && "$owner" != "$release" ]]; then
    echo "$resource is already owned by Helm release '$owner'; refusing to adopt it into '$release'." >&2
    exit 1
  fi
  if [[ -n "$current_namespace" && "$current_namespace" != "$NAMESPACE" ]]; then
    echo "$resource is already owned from namespace '$current_namespace'; refusing to adopt it." >&2
    exit 1
  fi
  if [[ -n "$managed_by" && "$managed_by" != "Helm" ]]; then
    echo "$resource has managed-by label '$managed_by'; refusing to change its owner." >&2
    exit 1
  fi

  kubectl label "$resource" --namespace "$NAMESPACE" \
    app.kubernetes.io/managed-by=Helm --overwrite >/dev/null
  kubectl annotate "$resource" --namespace "$NAMESPACE" \
    "meta.helm.sh/release-name=$release" \
    "meta.helm.sh/release-namespace=$NAMESPACE" --overwrite >/dev/null
}

# Adopt the resources from the previous kubectl deployment without replacing
# StatefulSets or their persistent volume claims.
adopt_resource service enerflux-postgres "$INFRA_RELEASE"
adopt_resource statefulset enerflux-postgres "$INFRA_RELEASE"
adopt_resource service enerflux-mqtt "$INFRA_RELEASE"
adopt_resource service enerflux-mqtt-headless "$INFRA_RELEASE"
adopt_resource statefulset enerflux-mqtt "$INFRA_RELEASE"
adopt_resource service enerflux-api "$APP_RELEASE"
adopt_resource deployment enerflux-api "$APP_RELEASE"
adopt_resource deployment enerflux-agent "$APP_RELEASE"

infra_args=(
  --set-string "postgres.image=$POSTGRES_IMAGE"
  --set-string "mqtt.image=$MQTT_IMAGE"
  --set-string "imagePullPolicy=$INFRA_IMAGE_PULL_POLICY"
  --set "mqtt.nodePort=$MQTT_NODE_PORT"
)

app_args=(
  --set-string "api.image.repository=$IMAGE_PREFIX/enerflux-api"
  --set-string "api.image.tag=$IMAGE_TAG"
  --set-string "agent.image.repository=$IMAGE_PREFIX/enerflux-agent"
  --set-string "agent.image.tag=$IMAGE_TAG"
  --set-string "imagePullPolicy=$IMAGE_PULL_POLICY"
  --set-string "migrations.image=$POSTGRES_IMAGE"
  --set-string "deploymentRevision=$DEPLOY_REVISION"
  --set "api.service.nodePort=$API_NODE_PORT"
)
if [[ -n "$IMAGE_PULL_SECRET_NAME" ]]; then
  infra_args+=(--set-string "imagePullSecrets[0].name=$IMAGE_PULL_SECRET_NAME")
  app_args+=(--set-string "imagePullSecrets[0].name=$IMAGE_PULL_SECRET_NAME")
fi
if [[ -n "$HELM_VALUES_FILE" ]]; then
  if [[ ! -f "$HELM_VALUES_FILE" ]]; then
    echo "Helm values file not found: $HELM_VALUES_FILE" >&2
    exit 1
  fi
  infra_args+=(--values "$HELM_VALUES_FILE")
  app_args+=(--values "$HELM_VALUES_FILE")
fi

helm upgrade --install "$INFRA_RELEASE" "$SCRIPT_DIR/helm/enerflux-infrastructure" \
  --namespace "$NAMESPACE" --create-namespace --wait --timeout "$ROLLOUT_TIMEOUT" \
  "${infra_args[@]}"

# The application chart runs the migration as a pre-install/pre-upgrade hook,
# after the infrastructure release is healthy and before app workloads change.
helm upgrade --install "$APP_RELEASE" "$SCRIPT_DIR/helm/enerflux" \
  --namespace "$NAMESPACE" --create-namespace --wait --timeout "$ROLLOUT_TIMEOUT" \
  "${app_args[@]}"

echo "EnerFlux is deployed in namespace '$NAMESPACE'."
echo "API: http://<k3s-node-ip>:$API_NODE_PORT"
echo "MQTT for LAN devices: <k3s-node-ip>:$MQTT_NODE_PORT"
