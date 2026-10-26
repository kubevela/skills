# Codex

## Skills

Codex reads user skills from `~/.agents/skills/` and repository skills from
`.agents/skills/` in the working tree. The older `~/.codex/skills/` path still appears in
some guides; do not rely on it.

```sh
git clone https://github.com/kubevela/skills.git
mkdir -p ~/.agents/skills
for s in kubevela-developer kubevela-sre kubevela-platform-engineer kubevela-defkit-author; do
  ln -sfn "$PWD/skills/$s" ~/.agents/skills/$s
done
```

Restart Codex and confirm all four appear in its skills list.

## Servers

Add to `~/.codex/config.toml`:

```toml
[mcp_servers.kubevela]
command = "vela"
args = ["mcp"]

[mcp_servers.kubernetes]
command = "kubernetes-mcp-server"
args = ["--config", "/etc/kubernetes-mcp-server/config.toml"]

[mcp_servers.grafana]
command = "mcp-grafana"
args = ["--disable-write"]
env = { GRAFANA_URL = "https://grafana.example.internal" }
```

The `kubevela` entry will not connect until `vela mcp` ships. The Kubernetes server's TOML
is in `../kubevela-sre/servers/kubernetes.md`.
