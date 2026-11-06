# Client setup

Steps for loading the skills into each client and connecting it to `vela mcp`. The
`kubevela` entry comes from `vela mcp config`, which prints it for the binary you run;
nothing in `vela` writes client configuration, so these pages, not a `vela` release, track
each client's format.

| Client | File | Skills location |
|---|---|---|
| Claude Code | `claude-code.md` | `~/.claude/skills/` or `.claude/skills/` in a repository |
| Codex | `codex.md` | `~/.agents/skills/` or `.agents/skills/` in a repository |
| Cursor | `cursor.md` | `~/.cursor/skills/` or `.cursor/skills/` in a repository |
| Gemini CLI | `gemini-cli.md` | `~/.gemini/skills/` or `.gemini/skills/` in a repository |

## What is available

- The four skills work today in the CLI-assisted profile: the agent runs `vela` and
  `kubectl` and follows the same procedure. Each skill states which guarantees that
  profile lacks.
- `vela mcp` is part of `vela` built from the `master` branch of
  github.com/kubevela/kubevela with `make vela-cli`; `vela mcp config` prints a working
  entry.
- `../kubevela-sre/references/servers.yaml` records the date each appendix was checked;
  every `version` there is empty, so no server release is pinned. Install the current release of each server and check its tool
  names against the appendix.

## Permissions every snippet assumes

- `vela mcp` starts read-only. `--allow-write` only where the mode policy has been
  reviewed. There is no resume tool; a person resumes a paused workflow with
  `vela workflow resume`.
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
