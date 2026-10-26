#!/usr/bin/env python3
"""Validate skill directories against the Agent Skills specification and this repository's
own rules.

    scripts/validate_skills.py [ROOT]

Checks, per skill directory (any top-level directory containing SKILL.md):
  - SKILL.md starts with a YAML frontmatter block closed by a second `---` line
  - the frontmatter parses as a YAML mapping
  - name: 1-64 chars, lowercase a-z, 0-9 and hyphens, no leading, trailing or consecutive
    hyphen, equal to the directory name
  - description: non-empty string, at most 1024 characters
  - compatibility (optional): string, 1-500 characters
  - metadata (optional): mapping of string keys to scalar values
  - allowed-tools (optional): string
  - SKILL.md body is at most 500 lines
  - every relative link or path reference to a file inside the skill resolves

Also validates kubevela-sre/references/servers.yaml: required fields per server and
that each `appendix` path exists.

Prints one line per problem and exits 1 if any were found. Requires PyYAML.
The upstream reference validator is `skills-ref validate <dir>`; this script exists so the
repository can also check cross-file references and the servers table.
"""
import re
import sys
from pathlib import Path

try:
    import yaml
except ImportError:  # pragma: no cover
    sys.exit("validate_skills.py needs PyYAML: pip install pyyaml")

NAME_RE = re.compile(r"^[a-z0-9]+(-[a-z0-9]+)*$")
LINK_RE = re.compile(r"\]\(([^)\s#]+)\)|`((?:scripts|references|servers|assets)/[^`\s]+)`")


def frontmatter(text):
    """Return (mapping or None, problem or None, body_lines)."""
    lines = text.split("\n")
    if not lines or lines[0].strip() != "---":
        return None, "SKILL.md does not start with a frontmatter delimiter", lines
    for i in range(1, len(lines)):
        if lines[i].strip() == "---":
            raw = "\n".join(lines[1:i])
            try:
                data = yaml.safe_load(raw)
            except yaml.YAMLError as e:
                return None, f"frontmatter is not valid YAML: {str(e).splitlines()[0]}", lines
            if not isinstance(data, dict):
                return None, "frontmatter is not a YAML mapping", lines
            return data, None, lines
    return None, "frontmatter has no closing delimiter", lines


def check_skill(skill_dir):
    problems = []
    skill = skill_dir.name
    path = skill_dir / "SKILL.md"
    text = path.read_text(encoding="utf-8")
    data, problem, lines = frontmatter(text)
    if problem:
        return [f"{skill}: {problem}"]

    name = data.get("name")
    if not isinstance(name, str) or not name:
        problems.append(f"{skill}: name is missing or not a string")
    else:
        if len(name) > 64:
            problems.append(f"{skill}: name is {len(name)} characters, limit 64")
        if not NAME_RE.match(name):
            problems.append(f"{skill}: name '{name}' must be lowercase letters, digits and single hyphens, not starting or ending with a hyphen")
        if name != skill:
            problems.append(f"{skill}: name '{name}' does not match the directory name")

    desc = data.get("description")
    if not isinstance(desc, str) or not desc.strip():
        problems.append(f"{skill}: description is missing or empty")
    elif len(desc) > 1024:
        problems.append(f"{skill}: description is {len(desc)} characters, limit 1024")

    compat = data.get("compatibility")
    if compat is not None and (not isinstance(compat, str) or not 1 <= len(compat) <= 500):
        problems.append(f"{skill}: compatibility must be a string of 1-500 characters")

    meta = data.get("metadata")
    if meta is not None:
        if not isinstance(meta, dict):
            problems.append(f"{skill}: metadata must be a mapping")
        else:
            for k, v in meta.items():
                if not isinstance(k, str) or isinstance(v, (dict, list)):
                    problems.append(f"{skill}: metadata entries must map string keys to scalar values (key {k!r})")

    tools = data.get("allowed-tools")
    if tools is not None and not isinstance(tools, str):
        problems.append(f"{skill}: allowed-tools must be a space-separated string")

    # Line count: count lines the way `wc -l` would, but also count a final line without a newline.
    line_count = len(lines) if text.endswith("\n") else len(lines)
    if text.endswith("\n"):
        line_count -= 1
    if line_count > 500:
        problems.append(f"{skill}: SKILL.md is {line_count} lines, limit 500")

    # Relative references inside the skill must resolve.
    for m in LINK_RE.finditer(text):
        ref = m.group(1) or m.group(2)
        if not ref or "://" in ref or ref.startswith("/") or ref.startswith("#"):
            continue
        if ref.startswith(("scripts/", "references/", "servers/", "assets/")):
            if not (skill_dir / ref).exists():
                problems.append(f"{skill}: references '{ref}' which does not exist in the skill directory")

    return problems


def check_servers(path):
    problems = []
    if not path.exists():
        return [f"{path}: missing"]
    try:
        data = yaml.safe_load(path.read_text(encoding="utf-8"))
    except yaml.YAMLError as e:
        return [f"{path}: not valid YAML: {str(e).splitlines()[0]}"]
    servers = (data or {}).get("servers")
    if not isinstance(servers, dict) or not servers:
        return [f"{path}: expected a non-empty 'servers' mapping"]
    for name, s in servers.items():
        if not isinstance(s, dict):
            problems.append(f"{path}: server '{name}' is not a mapping")
            continue
        for field in ("repo", "role", "transport", "appendix", "checked"):
            if field not in s:
                problems.append(f"{path}: server '{name}' lacks '{field}'")
        if s.get("transport") not in ("stdio", "http", None):
            problems.append(f"{path}: server '{name}' transport must be stdio or http")
        if "version" not in s:
            problems.append(f"{path}: server '{name}' must declare 'version' (null until evaluated)")
        appendix = s.get("appendix")
        if isinstance(appendix, str) and not (path.parent.parent / appendix).exists() and not (path.parent.parent.parent / appendix).exists():
            problems.append(f"{path}: server '{name}' appendix '{appendix}' does not exist")
    return problems


def main():
    root = Path(sys.argv[1]) if len(sys.argv) > 1 else Path(__file__).resolve().parent.parent
    problems = []
    skill_dirs = sorted(p for p in root.iterdir() if p.is_dir() and (p / "SKILL.md").exists())
    if not skill_dirs:
        problems.append(f"{root}: no skill directories found")
    for d in skill_dirs:
        problems.extend(check_skill(d))
    servers = root / "kubevela-sre" / "references" / "servers.yaml"
    if servers.exists() or (root / "kubevela-sre").exists():
        problems.extend(check_servers(servers))
    for p in problems:
        print(p)
    if problems:
        print(f"validate-skills: {len(problems)} problem(s)")
        return 1
    print(f"validate-skills: ok ({len(skill_dirs)} skills)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
