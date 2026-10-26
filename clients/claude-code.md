# Claude Code

## Skills

Install for every project:

```sh
git clone https://github.com/kubevela/skills.git
mkdir -p ~/.claude/skills
for s in kubevela-developer kubevela-sre kubevela-platform-engineer kubevela-defkit-author; do
  ln -sfn "$PWD/skills/$s" ~/.claude/skills/$s
done
```

Or scope to one repository by linking into `.claude/skills/` inside it. Claude Code picks
skills up on the next session start; `/skills` lists what it found.

## Servers

Add to `~/.claude.json` under `mcpServers`, or use `claude mcp add`:

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

The `kubevela` entry will not connect until `vela mcp` ships. The TOML for the Kubernetes
server is in `../kubevela-sre/servers/kubernetes.md`. Replace the Grafana entry with the
Datadog server's own setup if that is your observability backend. Verify inside a session
with `/mcp`.
