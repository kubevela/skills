# Datadog MCP server

Checked against: see `../references/servers.yaml` (`datadog`). Tool names below are from
the server's published tool list and must be re-verified whenever the pin moves.

Server: https://docs.datadoghq.com/bits_ai/mcp_server/

## Required configuration for the investigation profile

The Datadog MCP server authenticates through Datadog's own OAuth flow and exposes
read-only query tools. Use an application key scoped to read permissions on logs, APM,
metrics and monitors. Do not grant incident or monitor write scopes to the identity the
agent uses.

## Identity mapping

Never query by component name alone. Establish the mapping once per incident:

| Level | KubeVela handle | Typical Datadog tag through the Kubernetes integration |
|---|---|---|
| Cluster | `clusters[]` | `kube_cluster_name` |
| Namespace | `namespace` | `kube_namespace` |
| Application | `labels["app.oam.dev/name"]` and `labels["app.oam.dev/namespace"]` | `kube_app_oam_dev_name`, `kube_app_oam_dev_namespace`, or custom tags your integration maps |
| Component | `componentLabels[<component>]["app.oam.dev/component"]` | `kube_app_oam_dev_component` or a custom tag |
| Pod | `pods[].name` | `pod_name` |
| Service for traces | not emitted | `service`; usable only after the probe below finds spans |

Tag names depend on how the Kubernetes integration maps pod labels. Confirm with
`get_datadog_metric_context` before building queries, and record the mapping you found.
If a level is not present as a tag, say so in the report.

Nothing maps an Application to an APM service name. Probe with `search_datadog_spans` for
`kube_cluster_name:(<c1> OR <c2>) kube_namespace:<namespace>` and the pod names in the
window, with every cluster in `clusters`, and read the `service` of the spans returned. Use that service only if the probe returns spans; otherwise report traces
unavailable rather than guessing.

## Tools by intent

| Intent in the SRE sequence | Tool | Notes |
|---|---|---|
| Discover tag names | `get_datadog_metric_context` | Once per incident |
| Metrics for the window | `get_datadog_metric` | Latency and error-rate metrics scoped by the full tag set; pass `window.from` and `window.to` explicitly |
| Trace search and aggregation | `search_datadog_spans`, `aggregate_spans` | `service:<service> status:error` in the window, with the probe-confirmed service only; aggregate by resource to find the top failing endpoint |
| One trace | `get_datadog_trace` | After search has narrowed to a trace id |
| Logs behind the span | `search_datadog_logs`, `analyze_datadog_logs` | `kube_cluster_name:(<c1> OR <c2>) kube_namespace:<namespace> pod_name:(<pods[0].name> OR <pods[1].name>)`, one term per cluster and pod, plus a text filter for the failing target |
| What fired | `search_datadog_monitors`, `search_datadog_events` | First trigger time tightens the window |
| Incident context | `search_datadog_incidents`, `get_datadog_incident` | Only if an incident was declared |
| Recent changes | `get_change_stories` | Correlate with the revision's `appliedAt` |

Time windows: always pass the handles' `window.from` and `window.to`. Server defaults
are usually the last fifteen minutes and will miss the apply time.
