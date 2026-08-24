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
Azure SQL Database :1433 through Private Link
```

- The UI serves static files and proxies `/api/*` to the `backend` service.
- The backend exposes structured JSON logs, graceful shutdown, liveness and
  database-aware readiness endpoints.
- AKS uses the Microsoft Go SQL Server driver and Azure Workload Identity. No
  database username or password is stored in Kubernetes or Git.
- Docker Compose uses a disposable SQL Server Developer container and local-only
  `sa` password so the same T-SQL schema can be tested without Azure access.

## Repository layout

```text
apps/backend/                         Go API and hardened multi-stage Dockerfile
apps/ui/                              Static UI, NGINX proxy and Dockerfile
deploy/helm/pharmacy-platform/        Reusable Kubernetes Helm chart
deploy/helm/pharmacy-platform/templates/
.github/workflows/validate.yml        Go, container and Helm pull-request checks
compose.yaml                          Local UI/backend/SQL Server environment
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

Remove the local SQL Server volume only when a clean database is intended:

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
- The UI does not mount a Kubernetes API token. The backend receives only the
  projected, audience-scoped token required for workload identity.
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
- separate ServiceAccounts, including the workload-identity backend account;
- liveness, readiness, and startup probes;
- resource requests and limits compatible with namespace quotas;
- topology-spread preferences for future multi-node clusters;
- optional UI/backend PodDisruptionBudgets;
- non-secret Azure SQL server/database configuration; and
- least-connectivity NetworkPolicies for UI to backend, Azure SQL Private Link,
  DNS and the Entra token endpoint.

Development runs one UI and backend replica against `pharmacy-dev`.
Production-pattern values run two UI/backend replicas with PDBs against
`pharmacy-prod`. On the current one-node AKS lab, replicas and PDBs demonstrate
Kubernetes behavior but do not provide node-failure tolerance.

## Azure SQL identity contract

Terraform federates these exact Kubernetes identities:

```text
system:serviceaccount:dev:pharmacy-backend
system:serviceaccount:prod:pharmacy-backend
```

The environment-specific Helm release must set
`backend.workloadIdentity.clientId` to the corresponding Terraform output. The
AKS workload-identity webhook then injects `AZURE_CLIENT_ID`, `AZURE_TENANT_ID`
and a projected OIDC token into the backend pod. The Go driver exchanges that
token for an Azure SQL access token using `ActiveDirectoryWorkloadIdentity`.

The SQL hostname and database name are configuration rather than secrets. TLS
encryption and server-certificate validation are mandatory in Azure.

## Database migration contract

Normal backend startup checks connectivity but never modifies the schema. Run
the same container with the `migrate` argument to apply the idempotent T-SQL
schema and seed data:

```bash
/backend migrate
```

Docker Compose does this automatically for local development. AKS will use a
controlled migration Job and a separately authorized migration identity. Runtime
backend identities will receive only data read/write permissions.

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
