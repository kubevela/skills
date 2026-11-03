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

## The kubevela server

`vela mcp config` prints the entry for the `vela` binary you run it with, with its
absolute path and any server flags you add, such as `--namespace team-a` or
`--allow-write --direct-namespace dev`. Nothing is written for you; merge the printed `kubevela` entry into the `mcpServers` object of `~/.cursor/mcp.json` or a project's `.cursor/mcp.json`:

```sh
vela mcp config
```

Start the client from a shell pointed at the cluster you mean: the server uses the
kubeconfig and context of the client process that starts it.

## Companion servers

Add to `~/.cursor/mcp.json`, or to `.cursor/mcp.json` inside a project:

```json
{
  "mcpServers": {
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

The Kubernetes server's TOML is in `../kubevela-sre/servers/kubernetes.md`.
