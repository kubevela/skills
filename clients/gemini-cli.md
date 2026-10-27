# Gemini CLI

## Skills

Gemini CLI loads Agent Skills natively. User skills live in `~/.gemini/skills/`, workspace
skills in `.gemini/skills/` inside the repository, and the `.agents/skills/` alias is read
at both tiers.

```sh
git clone https://github.com/kubevela/skills.git
mkdir -p ~/.gemini/skills
for s in kubevela-developer kubevela-sre kubevela-platform-engineer kubevela-defkit-author; do
  ln -sfn "$PWD/skills/$s" ~/.gemini/skills/$s
done
```

In a session, `/skills list` shows what was discovered and `/skills reload` picks up
changes without restarting. Gemini CLI asks for consent before activating a skill.

## The kubevela server

`vela mcp config` prints the entry for the `vela` binary you run it with, with its
absolute path and any server flags you add, such as `--namespace team-a` or
`--allow-write --direct-namespace dev`. Nothing is written for you; merge the printed `kubevela` entry into the `mcpServers` object of `~/.gemini/settings.json`:

```sh
vela mcp config
```

Start the client from a shell pointed at the cluster you mean: the server uses the
kubeconfig and context of the client process that starts it.

## Companion servers

Add to `~/.gemini/settings.json`:

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
