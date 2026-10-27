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

## The kubevela server

`vela mcp config` prints the entry for the `vela` binary you run it with, with its
absolute path and any server flags you add, such as `--namespace team-a` or
`--allow-write --direct-namespace dev`. Nothing is written for you; put its `command` and `args` into a `[mcp_servers.kubevela]` table in `~/.codex/config.toml`:

```sh
vela mcp config
```

```toml
[mcp_servers.kubevela]
command = "/absolute/path/printed/by/vela"
args = ["mcp"]
```

Start Codex from a shell pointed at the cluster you mean: the server uses the kubeconfig
and context of the client process that starts it.

## Companion servers

Add to `~/.codex/config.toml`:

```toml
[mcp_servers.kubernetes]
command = "kubernetes-mcp-server"
args = ["--config", "/etc/kubernetes-mcp-server/config.toml"]

[mcp_servers.grafana]
command = "mcp-grafana"
args = ["--disable-write"]
env = { GRAFANA_URL = "https://grafana.example.internal" }
```

The Kubernetes server's TOML is in `../kubevela-sre/servers/kubernetes.md`.
