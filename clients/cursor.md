# Cursor

## Skills

Cursor reads personal skills from `~/.cursor/skills/` and project skills from
`.cursor/skills/` inside the repository. Each skill is a directory containing `SKILL.md`,
and the `name` field must match the directory name.

```sh
git clone https://github.com/kubevela/skills.git
mkdir -p ~/.cursor/skills
for s in kubevela-developer kubevela-sre kubevela-platform-engineer kubevela-defkit-author; do
  ln -sfn "$PWD/skills/$s" ~/.cursor/skills/$s
done
```

Open a new agent chat and type `/` to confirm the four skills are listed.

## Servers

Add to `~/.cursor/mcp.json`, or to `.cursor/mcp.json` inside a project:

```json
{
  "mcpServers": {
    "kubevela": {
      "command": "vela",
      "args": ["mcp"]
    },
    "kubernetes": {
      "command": "kubernetes-mcp-server",
      "args": ["--config", "/etc/kubernetes-mcp-server/config.toml"]
    },
    "grafana": {
      "command": "mcp-grafana",
      "args": ["--disable-write"],
      "env": {"GRAFANA_URL": "https://grafana.example.internal"}
    }
  }
}
```

The `kubevela` entry will not connect until `vela mcp` ships. The Kubernetes server's TOML
is in `../kubevela-sre/servers/kubernetes.md`.
