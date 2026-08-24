# Database identities and migrations

Azure SQL authentication uses Microsoft Entra workload identities. Kubernetes
Secrets, SQL passwords, and shared service accounts are not part of the Azure
deployment model.

| Identity | SQL permission boundary | Kubernetes account |
|---|---|---|
| Backend runtime | Read/write/execute on application objects | `pharmacy-backend` |
| Schema migration | Create/alter application schema and seed data | `pharmacy-migration` |
| Access bootstrap | Reconcile database users and role membership | Platform-owned temporary Job |

The runtime deployment cannot alter the schema. Argo CD orders the migration
ServiceAccount at sync wave `-2`, the completed migration Job at `-1`, and the
backend Deployment at `0`. The Job uses the exact backend image being promoted,
so migration code and application code remain version-compatible.

`migration.enabled` remains false until the GitOps release has the corresponding
Terraform output `migration_workload_identity_client_ids[environment]`. A
production delivery workflow should open a reviewed GitOps pull request that
sets both immutable image digests and the non-secret workload identity client
IDs. Application CI must not call `kubectl apply` directly.

The fixed Job name and Argo CD `Replace=true` sync option allow a new immutable
image revision to replace the completed previous migration Job. Migration SQL
must remain backward-compatible and idempotent because application replicas can
overlap during rolling deployment.
