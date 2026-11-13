# EnerFlux on k3s

This folder deploys the EnerFlux API and agent, PostgreSQL, and Mosquitto in one Kubernetes namespace. The API and MQTT broker use NodePorts so they can be reached without a separate ingress controller.

## Requirements

- A working k3s cluster and `kubectl` pointed at it
- Docker with Buildx, for building images
- `envsubst` and Python 3, for deployment
- A default StorageClass; k3s installations normally provide `local-path`
- The repository's root `.env` file

## Build and publish the application images

The scripts default to `ghcr.io/theabys` and the `develop` tag, matching the repository's image workflow. Override these values when using another registry or tag:

```bash
IMAGE_PREFIX=ghcr.io/your-owner IMAGE_TAG=develop ./k3s/build-images.sh
```

Buildx publishes multi-architecture `linux/amd64` and `linux/arm64` images. Set `PLATFORMS` to match a different cluster architecture. Log in to the registry before running the script if it requires authentication. Images must be pullable by the cluster nodes.

## Deploy

The deploy script reads the root `.env` without printing its values. It creates Kubernetes Secrets from that file and rewrites the database and MQTT broker URLs to in-cluster service addresses. PostgreSQL username, password, and database name are taken from `DATABASE_URL`.

```bash
IMAGE_PREFIX=ghcr.io/your-owner IMAGE_TAG=develop ./k3s/deploy.sh
```

By default, the API is available at `http://<k3s-node-ip>:30080`, and LAN devices can connect to the Mosquitto broker at `<k3s-node-ip>:31883`. Configure Shelly devices to publish to that broker. PostgreSQL is only exposed inside the cluster.

Useful overrides:

```bash
ENV_FILE=/path/to/.env NAMESPACE=enerflux IMAGE_TAG=develop ./k3s/deploy.sh
API_NODE_PORT=30081 ./k3s/deploy.sh
NAMESPACE=enerflux ./k3s/status.sh
```

`API_NODE_PORT` must be in Kubernetes' NodePort range (`30000`–`32767`). Open the chosen NodePorts in the host firewall if needed. Keep SolarLog and ETA URLs in `.env` reachable from pods; a URL using `localhost` refers to the pod itself.

The deploy script applies persistent volumes for PostgreSQL and Mosquitto and runs the SQL migrations before starting the API and agent. The first install uses the default StorageClass. The database password is initialized from the current `.env`; changing that URL later does not rotate the password in an already initialized PostgreSQL data volume.

If the GHCR packages are private, configure an image pull Secret in the `enerflux` namespace and attach it to that namespace's default ServiceAccount before deploying. The app images need to be pullable by each cluster node.
