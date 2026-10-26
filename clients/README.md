# Client setup

Snippets for loading the skills into each client and, once it ships, connecting to
`vela mcp`. `vela mcp setup` will write these for you later; until then, copy the snippet
for your client.

| Client | File | Skills location |
|---|---|---|
| Claude Code | `claude-code.md` | `~/.claude/skills/` or `.claude/skills/` in a repository |
| Codex | `codex.md` | `~/.agents/skills/` or `.agents/skills/` in a repository |
| Cursor | `cursor.md` | `~/.cursor/skills/` or `.cursor/skills/` in a repository |
| Gemini CLI | `gemini-cli.md` | `~/.gemini/skills/` or `.gemini/skills/` in a repository |

## What is and is not released

- The four skills work today in the CLI-assisted profile: the agent runs `vela` and
  `kubectl` and follows the same procedure. Each skill states which guarantees that
  profile lacks.
- `vela mcp` is a proposal. The server snippets below are shaped for it but there is no
  binary yet. Do not expect the `kubevela` server entry to connect.
- Third-party server versions are not yet pinned; see
  `../kubevela-sre/references/servers.yaml`. Until a version is recorded there, install
  the current release of each server and expect tool names in the appendices to need
  checking.

## Permissions every snippet assumes

- `vela mcp` starts read-only. `--allow-write` only where the mode policy has been
  reviewed; `--allow-resume` never outside a single-user sandbox.
- The Kubernetes MCP server runs with a TOML config setting `read_only = true` and
  denying Secret and RBAC kinds; see `../kubevela-sre/servers/kubernetes.md`. It is
  started with `--config`, not with flags.
- The Grafana MCP server starts with `--disable-write`. The Datadog server uses a
  read-scoped application key.

## Verifying discovery

After installing, open a new session and ask the client to list its skills (`/skills` in
Claude Code and Gemini CLI, the skills picker in Cursor and Codex). All four names should
appear. If one is missing, the directory name and the `name` field in its `SKILL.md`
probably differ; run `python3 scripts/validate_skills.py` from the repository root.
