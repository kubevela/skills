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

## The kubevela server

`vela mcp config` prints the entry for the `vela` binary you run it with, with its
absolute path and any server flags you add, such as `--namespace team-a` or
`--allow-write --direct-namespace dev`. Nothing is written for you; Claude Code's own command writes its configuration:

```sh
eval "claude mcp add --scope user kubevela -- $(vela mcp config --format command)"
```

`--format command` prints the server command shell-quoted, and `eval` is what keeps a
`vela` path with spaces in it as one argument. `--scope user` makes the server available
in every project; drop it to register it for the current repository only.

Claude Code starts the server with its own environment, so the cluster is whichever one
the shell that started Claude Code points at, not the one you ran `claude mcp add` from.
To pin a kubeconfig regardless of the shell, add `-e KUBECONFIG=/absolute/path` before
`kubevela` in the command.

## Companion servers

Add to `~/.claude.json` under `mcpServers`, or with `claude mcp add`:

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

The TOML for the Kubernetes server is in `../kubevela-sre/servers/kubernetes.md`. Replace the Grafana entry with the
Datadog server's own setup if that is your observability backend. Verify inside a session
with `/mcp`.
