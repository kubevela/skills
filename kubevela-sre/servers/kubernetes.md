# Kubernetes MCP server

Checked against: see `../references/servers.yaml` (`kubernetes`).

Server: https://github.com/containers/kubernetes-mcp-server

## Required configuration for the investigation profile

Upstream configures runtime behaviour through a TOML file, not CLI flags. The reference
configuration is **read-only**, which exposes only tools annotated as read-only.
`disable_destructive` is a different, weaker setting that still allows some writes; the
investigation profile does not use it.

```toml
# kubernetes-mcp-server --config /etc/kubernetes-mcp-server/config.toml
read_only = true

# Reads are not safe to disclose. Deny the kinds that carry credentials and access policy.
[[denied_resources]]
group = ""
version = "v1"
kind = "Secret"

[[denied_resources]]
group = "rbac.authorization.k8s.io"
version = "v1"
kind = "Role"

[[denied_resources]]
group = "rbac.authorization.k8s.io"
version = "v1"
kind = "RoleBinding"

[[denied_resources]]
group = "rbac.authorization.k8s.io"
version = "v1"
kind = "ClusterRole"

[[denied_resources]]
group = "rbac.authorization.k8s.io"
version = "v1"
kind = "ClusterRoleBinding"
```

Run it under a kubeconfig or ServiceAccount limited to the namespaces in scope for the
investigation. If the server you are connected to exposes any write tool, the setup is
wrong: report it and do not use it.

## Access policy

Same as the SRE skill, restated here so this file is complete on its own:

1. Prefer vela for anything `explain_application` listed as managed.
2. Read-only cross-checks through this server are allowed when vela's evidence is
   unavailable, stale or contradicted. Keep them to the namespace and objects in the
   handles, and record the fallback and its reason in the report.
3. Never mutate through this server.
4. Read-only is not safe-to-disclose. Do not quote secret values or customer data.

## Tools by intent

| Intent | Tool | Notes |
|---|---|---|
| Confirm the active context matches the handles | `configuration_view` | Before any other call |
| Node pressure or scheduling failures | `nodes_list`, `events_list` | Filter events by the namespace and pods from the handles; this is the right branch when pods never scheduled |
| Unmanaged objects in the namespace | `resources_list`, `resources_get` | Pass the cluster context from the handles when the server is multi-context |
| Pod state cross-check | `pods_list`, `pods_get` | Allowed under policy rule 2 when vela's view is stale or contradicted; record the fallback |
| Pod logs | `pods_log` | Allowed under policy rule 2 only when no observability server is loaded; otherwise use that server |

Tool names vary slightly between releases of this server. If a name here does not exist,
list the server's tools and pick the equivalent rather than guessing arguments. Report
the substitution.
