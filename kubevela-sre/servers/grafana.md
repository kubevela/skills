# Grafana MCP server

Checked against: see `../references/servers.yaml` (`grafana`). Tool names below were read
from the upstream server's tool table and must be re-verified whenever the pin moves.

Server: https://github.com/grafana/mcp-grafana

## Required configuration for the investigation profile

Start the server with `--disable-write`. The server exposes dashboard, datasource,
alerting and other write tools by default; the investigation profile must not have them.
Scope the service account to Viewer-level read actions on the datasources you query.

## Identity mapping

Never query by component name alone. Two Applications in different namespaces can reuse
a component name, and a query that matches both is misleading evidence. Establish the
mapping once per incident and reuse it:

| Level | KubeVela handle | Typical telemetry label after relabeling |
|---|---|---|
| Cluster | `clusters[]` | `cluster` |
| Namespace | `namespace` | `namespace` |
| Application | `metricLabels["app_oam_dev_name"]` | `app_oam_dev_name` |
| Component | `componentLabels[<component>]["app.oam.dev/component"]` | `app_oam_dev_component` |
| Pod | `pods[].name` | `pod` |

Prometheus relabels dots and slashes to underscores, so the KubeVela labels usually
appear as above; the handles' `metricLabels` already carry that form. Confirm with one `query_prometheus` instant query before building the
range query. If a level is not retained by the backend, say so in the report.

## Tools by intent

| Intent in the SRE sequence | Tool | Notes |
|---|---|---|
| Find datasources | `list_datasources` | Once per session; remember the UIDs |
| Metrics for the window | `query_prometheus` | Range query over `window.from` to `window.to` with the full identity selector, for example `{namespace="<ns>", app_oam_dev_name="<app>", app_oam_dev_component="<component>", cluster="<cluster>"}` |
| Log volume before reading lines | `query_loki_stats` | Cheap check that the selector matches anything in the window |
| Logs behind the span | `query_loki_logs` | Stream selector built from `metricLabels` and the namespace, for example `{namespace="team-a", app_oam_dev_name="shop"}`; line filter for the failing span's target |
| Trace search | `search_tempo_traces` | By the service name your tracing uses for the Application, error status, within `window` |
| Trace query | `query_tempo` | TraceQL for a specific span pattern once search has narrowed it |
| Alert context | `list_alert_rules`, `get_alert_rule_by_uid` | Confirm what fired and when; use the first trigger time to tighten the window |
| Dashboards | `search_dashboards` | Links for the report, not evidence |

If a tool name here is absent from the server you are connected to, list its tools and
pick the equivalent rather than guessing arguments. Report the substitution.
