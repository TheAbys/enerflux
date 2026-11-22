# EnerFlux on k3s

This folder deploys the EnerFlux API and agent, PostgreSQL, and Mosquitto in one Kubernetes namespace. The API and MQTT broker use NodePorts so they can be reached without a separate ingress controller.

## Requirements

- A working k3s cluster with `kubectl` configured for it
- Helm 4.3.1 and Python 3, for deployment
- Docker with Buildx, for building images. The image script creates a multi-platform `docker-container` builder named `enerflux-multiarch` if needed; override it with `BUILDER_NAME`.
- A default StorageClass; k3s installations normally provide `local-path`
- The repository's root `.env` file

## Build and publish the application images

The scripts default to `ghcr.io/theabys` and the `develop` tag, matching the repository's image workflow. Override these values when using another registry or tag:

```bash
IMAGE_PREFIX=ghcr.io/your-owner IMAGE_TAG=develop ./k3s/build-images.sh
```

Buildx publishes multi-architecture `linux/amd64` and `linux/arm64` images. Set `PLATFORMS` to match a different cluster architecture. Log in to the registry before running the script if it requires authentication. Images must be pullable by the cluster nodes.

Use a version tag such as `IMAGE_TAG=1.2.3` for a stable deployment. The default `develop` tag follows the development branch and can change on every push.

The GitHub Actions workflow uses the same target platforms. It builds both images on pull requests without publishing them, then publishes images for `develop` and version tags. The API image includes the Vite-built frontend assets; the agent image does not need the Node build stage.

Dependabot checks the root Dockerfile, Compose images, Helm chart images, and GitHub Actions weekly. PostgreSQL stays on major version 16 for the existing persistent data; major-version upgrades need a separate migration plan.

## Deploy

The deploy script reads the root `.env` without printing its values. It derives PostgreSQL credentials from `DATABASE_URL` and sets its host to the in-cluster database service. The application chart sets the MQTT broker URL to the in-cluster Mosquitto service. Secrets and the shared ConfigMap are applied by the script; Helm manages the workloads and services.

```bash
IMAGE_PREFIX=ghcr.io/your-owner IMAGE_TAG=develop ./k3s/deploy.sh
```

By default, the API is available at `http://<k3s-node-ip>:30080`, and LAN devices can connect to the Mosquitto broker at `<k3s-node-ip>:31883`. Configure Shelly devices to publish to that broker. PostgreSQL is only exposed inside the cluster.

The script installs two Helm releases in order. The infrastructure chart provisions PostgreSQL and Mosquitto, and waits for them to become ready. The application chart then runs SQL migrations as a `pre-install`/`pre-upgrade` hook before updating the API and agent. Splitting the charts ensures the database is ready before migrations start.

Existing resources from the previous kubectl deployment are adopted by the matching Helm release. The fixed StatefulSet names and volume claim templates preserve the existing database and MQTT PVCs. Do not uninstall the infrastructure release if you need to keep its PVCs.

Useful overrides:

```bash
ENV_FILE=/path/to/.env NAMESPACE=enerflux IMAGE_TAG=develop ./k3s/deploy.sh
API_NODE_PORT=30081 ./k3s/deploy.sh
MQTT_NODE_PORT=31884 ./k3s/deploy.sh
IMAGE_PULL_SECRET_NAME=ghcr-auth ./k3s/deploy.sh
HELM_VALUES_FILE=/path/to/enerflux-values.yaml ./k3s/deploy.sh
NAMESPACE=enerflux ./k3s/status.sh
```

`API_NODE_PORT` and `MQTT_NODE_PORT` must be in Kubernetes' NodePort range (`30000`–`32767`). Open the chosen NodePorts in the host firewall if needed. The named image pull Secret must already exist in the target namespace. The optional Helm values file is applied to both charts; chart-specific values are ignored by the other chart. Use it to override storage sizes, resource limits, or chart settings. Keep SolarLog and ETA URLs in `.env` reachable from pods; a URL using `localhost` refers to the pod itself.

The charts create persistent volumes for PostgreSQL and Mosquitto. The first install uses the default StorageClass unless a storage class is configured in the Helm values. Each deploy creates a new app revision, so mutable `develop` tags and changed secrets are picked up. API pods roll with a ready replacement; the single agent uses a recreate strategy to avoid two collectors running at once. The API readiness probe checks PostgreSQL, while liveness only checks the API process.

A temporary database outage therefore removes the API from Service traffic without triggering a restart loop. PostgreSQL and the API have startup probes to allow time for first-time initialization.

The database has one replica. Its PVC survives pod restarts, but the default k3s `local-path` storage does not protect data from node or disk loss. Back up the database regularly and keep a copy off the cluster. The scripts are manual helpers; schedule them from a trusted host and sync the output off-node. They write a compressed dump to `backups/` by default; set `BACKUP_DIR` to another mounted or synced location:

```bash
BACKUP_DIR=/mnt/backups/enerflux ./k3s/backup-postgres.sh
```

The dump excludes object ownership and grants. Store the `.env` file securely as well; it contains the database credentials needed to recreate the deployment.

To restore, stop writes by scaling down the API and agent, then run the guarded restore command:

```bash
kubectl scale --namespace enerflux deployment/enerflux-api deployment/enerflux-agent --replicas=0
NAMESPACE=enerflux BACKUP_FILE=/mnt/backups/enerflux-postgres-<timestamp>.dump CONFIRM_RESTORE=yes ./k3s/restore-postgres.sh
IMAGE_TAG=develop ./k3s/deploy.sh
```

The restore script refuses to run unless both application deployments are scaled to zero. Keep a verified backup before database migrations or major version upgrades.

The database password is initialized from the current `.env`; changing that URL later does not rotate the password in an already initialized PostgreSQL data volume.

If the GHCR packages are private, create an image pull Secret in the deployment namespace and set `IMAGE_PULL_SECRET_NAME` when deploying. The app images need to be pullable by each cluster node.
