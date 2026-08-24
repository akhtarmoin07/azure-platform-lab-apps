# Azure Platform Lab Applications

This repository contains the deliberately small application workload used to
exercise the Azure platform. Business logic is kept simple so containerization,
Helm, GitOps, identity, networking, observability, reliability, and promotion
remain visible.

## Architecture

```text
Browser
  |
  v
NGINX UI :8080
  |  reverse proxies /api
  v
Go backend :8080
  |
  v
PostgreSQL :5432
```

- The UI serves static files and proxies `/api/*` to the `backend` service.
- The backend exposes structured JSON logs, graceful shutdown, liveness and
  database-aware readiness endpoints.
- PostgreSQL is used by Docker Compose and the first in-cluster lab release.
  The Helm chart accepts an existing database Secret so Azure Database for
  PostgreSQL can replace the StatefulSet without changing application images.

## Repository layout

```text
apps/backend/                         Go API and hardened multi-stage Dockerfile
apps/ui/                              Static UI, NGINX proxy and Dockerfile
deploy/helm/pharmacy-platform/        Reusable Kubernetes Helm chart
deploy/helm/pharmacy-platform/templates/
.github/workflows/validate.yml        Go, container and Helm pull-request checks
compose.yaml                          Local UI/backend/PostgreSQL environment
.env.example                          Non-secret local environment template
```

## Local development

Create the ignored local environment file:

```bash
cp .env.example .env
```

Replace the example password, then start the stack:

```bash
docker compose up --build
```

Open <http://localhost:8080>. Useful endpoints are:

```text
GET /healthz              UI liveness
GET /health/live          Backend process liveness
GET /health/ready         Backend and database readiness
GET /api/medicines        Example database-backed response
```

Stop the containers without removing database data:

```bash
docker compose down
```

Remove the local PostgreSQL volume only when a clean database is intended:

```bash
docker compose down --volumes
```

## Backend dependency lock

The backend pins its direct dependency in `go.mod`. Generate and commit the
transitive checksum file before opening the first pull request:

```bash
cd apps/backend
go mod tidy
go test ./...
cd ../..
```

`go.sum` is required for reproducible CI and container builds.

## Container security

- Both application containers run as non-root numeric users.
- The Go binary is compiled with CGO disabled in a multi-stage build.
- Kubernetes drops Linux capabilities and prevents privilege escalation.
- UI and backend root filesystems are read-only in Kubernetes.
- No Kubernetes service account token is mounted into these pods.
- Images and dependencies are pinned to explicit version families rather than
  floating without any version.

Image scanning, SBOM generation, signing, and ACR publishing are added in the
next CI layer.

## Helm chart

Validate and render both environments without touching a cluster:

```bash
helm lint deploy/helm/pharmacy-platform

helm template pharmacy-dev deploy/helm/pharmacy-platform \
  --namespace dev \
  --values deploy/helm/pharmacy-platform/values-dev.yaml

helm template pharmacy-prod deploy/helm/pharmacy-platform \
  --namespace prod \
  --values deploy/helm/pharmacy-platform/values-prod.yaml
```

The chart creates:

- UI and backend Deployments and ClusterIP Services;
- separate ServiceAccounts without automatic API tokens;
- liveness, readiness, and startup probes;
- resource requests and limits compatible with namespace quotas;
- topology-spread preferences for future multi-node clusters;
- optional UI/backend PodDisruptionBudgets;
- an optional PostgreSQL StatefulSet and headless Service;
- least-connectivity NetworkPolicies for UI → backend → database.

Development runs one UI and backend replica with ephemeral database storage.
Production-pattern values run two UI/backend replicas, enable PDBs, and request
a PostgreSQL persistent volume. On the current one-node AKS lab, replicas and
PDBs demonstrate Kubernetes behavior but do not provide node-failure tolerance.

## Secret contract

The chart never generates or stores a database password in Git. Before a Helm or
Argo deployment, the destination namespace must contain a Secret named
`pharmacy-database` with:

```text
postgres-password    PostgreSQL password
database-url         Complete backend PostgreSQL connection string
```

For local Kubernetes troubleshooting only, this can be created imperatively.
The target implementation will source these values from Azure Key Vault through
AKS workload identity and the Secrets Store CSI driver.

## Delivery model

1. Application CI tests code, builds containers, scans/signs them, and pushes
   immutable images to ACR.
2. CI proposes an image digest change to the GitOps repository.
3. Argo CD reconciles the dev Application.
4. Verification promotes the same digest—not a rebuild—to prod through a
   reviewed pull request.
5. Argo CD deploys prod within the `prod` AppProject and namespace boundary.

Application CI does not directly call `kubectl apply`; Git remains the deployment
source of truth.
