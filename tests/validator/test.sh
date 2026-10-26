#!/usr/bin/env bash
# Negative and positive tests for scripts/validate_skills.py. Builds throwaway skill trees
# in a temp directory and asserts the validator's verdict. Exits 1 if any check fails.
set -uo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
validator="$here/../../scripts/validate_skills.py"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
fail=0

# make_skill <root> <dir> <frontmatter> [body-lines]
make_skill() {
  mkdir -p "$1/$2" "$1/kubevela-sre/references"
  printf '%s\n' "$3" > "$1/$2/SKILL.md"
  if [ -n "${4:-}" ]; then seq 1 "$4" | sed 's/^/line /' >> "$1/$2/SKILL.md"; fi
  cat > "$1/kubevela-sre/references/servers.yaml" <<'EOF'
servers:
  kubernetes:
    repo: https://example.org
    role: test
    transport: stdio
    version: null
    appendix: kubevela-sre/servers/kubernetes.md
    checked: 2026-01-01
EOF
  mkdir -p "$1/kubevela-sre/servers" && touch "$1/kubevela-sre/servers/kubernetes.md"
}

expect() { # expect <label> <accept|reject> <root>
  if python3 "$validator" "$3" >/dev/null 2>&1; then verdict=accept; else verdict=reject; fi
  if [ "$verdict" = "$2" ]; then echo "PASS $1"; else echo "FAIL $1: expected $2, got $verdict"; python3 "$validator" "$3" | sed 's/^/    /'; fail=1; fi
}

good=$'---\nname: kubevela-good\ndescription: A valid skill. Use when testing.\n---\n# Body'
r="$tmp/good"; make_skill "$r" kubevela-good "$good"; expect "valid skill accepted" accept "$r"

r="$tmp/quoted"; make_skill "$r" kubevela-quoted $'---\nname: "kubevela-quoted"\ndescription: "Quoted values are valid YAML."\n---\n# Body'; expect "quoted YAML values accepted" accept "$r"

r="$tmp/emptydesc"; make_skill "$r" kubevela-emptydesc $'---\nname: kubevela-emptydesc\ndescription: ""\n---\n# Body'; expect "empty quoted description rejected" reject "$r"

r="$tmp/noclose"; make_skill "$r" kubevela-noclose $'---\nname: kubevela-noclose\ndescription: x\n# Body without closing delimiter'; expect "missing closing delimiter rejected" reject "$r"

r="$tmp/badyaml"; make_skill "$r" kubevela-badyaml $'---\nname: kubevela-badyaml\ndescription: [unclosed\n---\n# Body'; expect "malformed YAML rejected" reject "$r"

long="$(printf 'x%.0s' $(seq 1 1030))"
r="$tmp/longdesc"; make_skill "$r" kubevela-longdesc $'---\nname: kubevela-longdesc\ndescription: >\n  '"$long"$'\n---\n# Body'; expect "block-scalar description over 1024 rejected" reject "$r"

r="$tmp/trailing"; make_skill "$r" kubevela-trailing- $'---\nname: kubevela-trailing-\ndescription: x\n---\n# Body'; expect "trailing hyphen rejected" reject "$r"

r="$tmp/double"; make_skill "$r" kubevela--double $'---\nname: kubevela--double\ndescription: x\n---\n# Body'; expect "consecutive hyphens rejected" reject "$r"

r="$tmp/mismatch"; make_skill "$r" kubevela-dirname $'---\nname: kubevela-other\ndescription: x\n---\n# Body'; expect "name not matching directory rejected" reject "$r"

r="$tmp/toolong"; make_skill "$r" kubevela-toolong "$good" 500; expect "501+ line SKILL.md rejected" reject "$r"
r="$tmp/toolong2"; make_skill "$r" kubevela-toolong2 "$good" 500; printf 'no newline' >> "$r/kubevela-toolong2/SKILL.md"; expect "over-limit file without final newline rejected" reject "$r"

r="$tmp/badref"; make_skill "$r" kubevela-badref $'---\nname: kubevela-badref\ndescription: x\n---\nSee [guide](references/missing.md).'; expect "dangling reference rejected" reject "$r"

r="$tmp/goodref"; make_skill "$r" kubevela-goodref $'---\nname: kubevela-goodref\ndescription: x\n---\nSee [guide](references/guide.md).'; mkdir -p "$r/kubevela-goodref/references"; touch "$r/kubevela-goodref/references/guide.md"; expect "resolving reference accepted" accept "$r"

r="$tmp/badmeta"; make_skill "$r" kubevela-badmeta $'---\nname: kubevela-badmeta\ndescription: x\nmetadata:\n  nested:\n    too: deep\n---\n# Body'; expect "nested metadata rejected" reject "$r"

r="$tmp/servers"; make_skill "$r" kubevela-servers "$good"; sed -i 's/appendix: .*/appendix: kubevela-sre\/servers\/nope.md/' "$r/kubevela-sre/references/servers.yaml"; expect "servers.yaml with missing appendix rejected" reject "$r"

exit $fail
