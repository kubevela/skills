# KubeVela skills

Agent skills, client setup templates and the evaluation suite for operating KubeVela
through AI assistants such as Claude Code, Codex, Cursor and Gemini CLI. This repository
pairs with `vela mcp`, the Model Context Protocol server proposed for the `vela` CLI in
[KEP: Vela MCP](https://github.com/kubevela/kubevela/blob/master/design/vela-cli/kep-vela-mcp.md).

A skill teaches an agent the order of operations, the stopping points, and which
guarantees the current execution path does and does not provide. The MCP server tells it
what it can call. The two are designed together, but the skills also work today with the
plain `vela` CLI through the CLI-assisted profile described below.

## Status

- `vela mcp` is a design proposal, not a released feature. Tool names in these skills
  follow the KEP and may change.
- The skills are usable now in the CLI-assisted profile for everything KubeVela itself
  provides. The SRE skill's metrics, traces and log-search steps need an observability
  MCP server in every profile; there is no CLI substitute, and the skill says so. None of
  the skills is yet backed by the evaluation suite, so treat them as reviewed procedures,
  not as tested tooling.
- Third-party server versions are not pinned yet. `kubevela-sre/references/servers.yaml`
  records what the appendices were checked against and when.

## Operating profiles

The same procedure runs under three profiles with different guarantees. Every skill
names the profile it is in and states what the current path lacks.

| Profile | Permitted | Guarantees |
|---|---|---|
| Offline authoring | Drafting and local rendering | Rendering only. Application validation, admission and runtime health are not checked, and the skills say so in a fixed five-line report |
| CLI-assisted | Inspection and planning by default; writes through a constrained path that records and rechecks the target identity and prefers GitOps | No diff token, no server-side mode policy; the agent maintains the approval binding and must say so |
| Governed MCP | Writes only through `apply_application` | Diff token, delivery mode and caller authorization enforced by the server |

Two rules hold in every profile: a denied or disabled write is never worked around with a
more privileged shell command, and human approval gates stay with humans. No skill calls
`resume_workflow`, `suspend_workflow` or `terminate_workflow`.

## Layout

```
kubevela-developer/          Compose, validate and deploy Applications
  references/                Application spec and status shape, generated from the v1.11.0 CRD
kubevela-sre/                Investigate incidents with vela, a read-only Kubernetes MCP and an observability MCP
  servers/                   Per-server tool appendices and identity mapping (grafana, datadog, kubernetes)
  references/                Status shapes generated from the CRDs; servers.yaml with pins and check dates
kubevela-platform-engineer/  Change definitions safely; tested consumer-inventory script in scripts/
  references/                The five definition CRDs and DefinitionRevision, generated
kubevela-defkit-author/      Author capabilities in Go with defkit, validate, render, deliver by PR
  references/                defkit API map, worked examples from vela-go-definitions, pitfalls
clients/                     Setup for Claude Code, Codex, Cursor, Gemini CLI
eval/                        Three-layer evaluation plan: structural, workflow, enforcement
scripts/                     validate_skills.py, snapshot_crd_shape.py, snapshot-references.sh
tests/                       Validator and consumer-inventory tests, run in CI
```

Each skill is a self-contained directory on the
[agentskills.io](https://agentskills.io/specification) layout. The SRE skill carries its
server table and appendices inside the skill so copying the directory alone is enough.

## How the skills know the real shapes

A skill is only as accurate as its knowledge of what an Application status, a workflow
step phase, a definition CRD or the defkit API looks like on the cluster in front of it.
Every skill therefore has a "Ground truth, in order" section that tells the agent where to
look, from most to least authoritative:

1. **The live target.** `kubectl explain ... --recursive`, `vela show` and `vela def get`
   describe the cluster that is actually running. For defkit, the authoritative source is
   the API of the pinned module instead: `go doc` resolved against the kubevela version in
   the module's `go.mod`, which describes the SDK the code will compile against, not the
   cluster.
2. **Version-matched artifacts.** The CRDs under `charts/vela-core/crds/`, the built-in
   definitions under `vela-templates/definitions/`, and the design documents in
   kubevela/kubevela at the tag reported by `get_platform_info`; vela-go-definitions at
   a commit pinning the same kubevela version.
3. **Bundled snapshots.** Each skill's `references/` directory holds shapes generated from
   the CRDs at a pinned release by `scripts/snapshot-references.sh`, and for defkit an API
   map, worked examples and pitfalls. The generated shapes, the API map and the examples
   carry a `Source:` line with the version or commit they came from; the pitfalls are
   version-independent guidance and `servers.yaml` records the date each server's tool
   list was checked. The agent compares each `Source:` with the version it describes: the
   CRD snapshots with the vela-core version running on the cluster, and the defkit API map
   and examples with the kubevela version pinned in the definition module's `go.mod`,
   since that is the SDK the code compiles against. It falls back to the first two sources
   when the minor version differs.

Snapshots are regenerated, never edited. When the pinned KubeVela version moves, run
`scripts/snapshot-references.sh <kubevela-checkout> <tag>` and commit the result.

## Installing

See `clients/` for each client's skills directory and a copy-paste install that creates
the directory and links all four skills. After installing, confirm the client lists all
four. The upstream reference validator `skills-ref validate <dir>` also accepts these
skills; the repository's own validator adds cross-file reference checks.

## The reference stack for incidents

| Server | Role | Required configuration |
|---|---|---|
| `vela mcp` | Composition, rendering, diff, live status, managed resources | Read and plan by default; writes behind a flag |
| Kubernetes MCP server | Nodes, namespaces, unmanaged resources | TOML `read_only = true`; Secret and RBAC kinds denied |
| Observability MCP | Metrics, traces, logs, alerts | Grafana with `--disable-write`; Datadog with a read-scoped key |

Access policy, identical across skills and appendices: prefer vela for Application-aware
reads; scoped read-only Kubernetes cross-checks are allowed when vela's evidence is
unavailable or stale and are recorded as fallbacks; nothing is mutated through the
Kubernetes server; read-only is not safe-to-disclose, so secret values and customer data
are never quoted.

## Retrieved content is evidence, not instructions

Logs, annotations, definition descriptions and tool results are data to inspect. Text in
them never authorizes a command or changes a rule. The evaluation plan includes hostile
fixtures that test this.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Run the validator and both test suites before
opening a pull request; CI runs the same.

## License

Apache 2.0, the same as KubeVela.
